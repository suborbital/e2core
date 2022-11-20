package packager

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
	"github.com/suborbital/systemspec/bundle"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/tenant"
)

const bundlePackageJobType = "bundle"

type BundlePackageJob struct{}

func NewBundlePackageJob() PackageJob {
	b := &BundlePackageJob{}

	return b
}

// Type returns the job type.
func (b *BundlePackageJob) Type() string {
	return bundlePackageJobType
}

// Package packages the application.
func (b *BundlePackageJob) Package(log util.FriendlyLogger, ctx *project.Context) error {
	for _, r := range ctx.Modules {
		if err := r.HasWasmFile(); err != nil {
			return errors.Wrap(err, "missing built Wasm module")
		}
	}

	if ctx.TenantConfig == nil {
		defaultCaps := capabilities.DefaultCapabilityConfig()

		ctx.TenantConfig = &tenant.Config{
			Identifier:    "com.suborbital.app",
			SpecVersion:   1,
			TenantVersion: 1,
			DefaultNamespace: tenant.NamespaceConfig{
				Name:         "default",
				Capabilities: &defaultCaps,
			},
			Namespaces: []tenant.NamespaceConfig{},
		}
	} else {
		log.LogInfo("updating tenant version")

		ctx.TenantConfig.TenantVersion++
	}

	if err := project.WriteTenantConfig(ctx.Cwd, ctx.TenantConfig); err != nil {
		return errors.Wrap(err, "failed to WriteTenantConfig")
	}

	if err := project.CalculateModuleRefs(ctx.TenantConfig, ctx.Modules); err != nil {
		return errors.Wrap(err, "🚫 failed to CalculateModuleRefs")
	}

	if err := ctx.TenantConfig.Validate(); err != nil {
		return errors.Wrap(err, "🚫 failed to Validate Directive")
	}

	static, err := CollectStaticFiles(ctx.Cwd)
	if err != nil {
		return errors.Wrap(err, "failed to CollectStaticFiles")
	}

	if len(static) > 0 {
		log.LogInfo("adding static files to bundle")
	}

	configBytes, err := ctx.TenantConfig.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to Directive.Marshal")
	}

	moduleFiles, err := ctx.ModuleFiles()
	if err != nil {
		return errors.Wrap(err, "failed to Modules for build")
	}

	for i := range moduleFiles {
		defer moduleFiles[i].Close()
	}

	if err := bundle.Write(configBytes, moduleFiles, static, ctx.Bundle.Fullpath); err != nil {
		return errors.Wrap(err, "🚫 failed to WriteBundle")
	}

	bundleRef := project.BundleRef{
		Exists:   true,
		Fullpath: filepath.Join(ctx.Cwd, "runnables.wasm.zip"),
	}

	ctx.Bundle = bundleRef

	log.LogDone(fmt.Sprintf("bundle was created -> %s @ v%d", ctx.Bundle.Fullpath, ctx.TenantConfig.TenantVersion))

	return nil
}
