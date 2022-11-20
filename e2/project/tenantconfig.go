package project

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/systemspec/fqmn"
	"github.com/suborbital/systemspec/tenant"
)

// WriteTenantConfig writes a tenant config to disk.
func WriteTenantConfig(cwd string, cfg *tenant.Config) error {
	filePath := filepath.Join(cwd, "tenant.json")

	configBytes, err := cfg.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to Marshal")
	}

	if err := ioutil.WriteFile(filePath, configBytes, util.PermFilePrivate); err != nil {
		return errors.Wrap(err, "failed to WriteFile")
	}

	return nil
}

// readTenantConfig finds a tenant.json from disk but does not validate it.
func readTenantConfig(cwd string) (*tenant.Config, error) {
	filePath := filepath.Join(cwd, "tenant.json")

	tenantBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadFile for Directive")
	}

	t := &tenant.Config{}
	if err := t.Unmarshal(tenantBytes); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal Directive")
	}

	return t, nil
}

// readQueriesFile finds a queries.yaml from disk.
func readQueriesFile(cwd string) ([]tenant.DBQuery, error) {
	filePath := filepath.Join(cwd, "Queries.yaml")

	configBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to ReadFile for Queries.yaml")
	}

	t := &tenant.Config{}
	if err := t.UnmarshalYaml(configBytes); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal Directive")
	}

	return t.DefaultNamespace.Queries, nil
}

// readConnectionsFile finds a queries.yaml from disk.
func readConnectionsFile(cwd string) ([]tenant.Connection, error) {
	filePath := filepath.Join(cwd, "Connections.yaml")

	configBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to ReadFile for Queries.yaml")
	}

	t := &tenant.Config{}
	if err := t.UnmarshalYaml(configBytes); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal Directive")
	}

	return t.DefaultNamespace.Connections, nil
}

// CalculateModuleRefs calculates the hash refs for all modules and validates correctness of the config.
func CalculateModuleRefs(cfg *tenant.Config, mods []ModuleDir) error {
	dirModules := make([]tenant.Module, len(mods))

	// for each module, calculate its ref (a.k.a. its hash), and then add it to the context.
	for i := range mods {
		mod := mods[i]
		modFile, err := mod.WasmFile()
		if err != nil {
			return errors.Wrap(err, "failed to WasmFile")
		}

		defer modFile.Close()

		hash, err := calculateModuleRef(modFile)
		if err != nil {
			return errors.Wrap(err, "failed to calculateModuleRef")
		}

		mod.Module.Ref = hash
		rev := tenant.ModuleRevision{
			Ref: hash,
		}

		if mod.Module.Revisions == nil {
			mod.Module.Revisions = []tenant.ModuleRevision{rev}
		} else {
			mod.Module.Revisions = append(mod.Module.Revisions, rev)
		}

		FQMN, err := fqmn.FromParts(cfg.Identifier, mod.Module.Namespace, mod.Module.Name, hash)
		if err != nil {
			return errors.Wrap(err, "failed to fqmn.FromParts")
		}

		mod.Module.FQMN = FQMN

		dirModules[i] = *mod.Module
	}

	// now that refs are calculated, ensure that all modules referenced
	// in the tenant's workflows are present in the tenant's module list
	workflowMods := getWorkflowFQMNList(cfg)

	missing := []string{}

	for _, modFQMN := range workflowMods {
		FQMN, err := fqmn.Parse(modFQMN)
		if err != nil {
			return errors.Wrapf(err, "failed to parse FQMN %s", modFQMN)
		}

		found := false

		for _, dirMod := range dirModules {
			if dirMod.Name == FQMN.Name && dirMod.Namespace == FQMN.Namespace {
				found = true
				break
			}
		}

		if !found {
			missing = append(missing, modFQMN)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("the following modules referenced in workflows were not found: %s", strings.Join(missing, ", "))
	}

	cfg.Modules = dirModules

	return cfg.Validate()
}

// calculateModuleRef calculates the hex-encoded sha256 hash of a module file.
func calculateModuleRef(mod io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, mod); err != nil {
		return "", errors.Wrap(err, "failed to Copy module contents")
	}

	hashBytes := hasher.Sum(nil)

	return hex.EncodeToString(hashBytes), nil
}

// getWorkflowFQMNList gets a full list of all functions used in the config's workflows.
func getWorkflowFQMNList(cfg *tenant.Config) []string {
	modMap := map[string]bool{}

	// collect all the workflows in all of the namespaces.
	workflows := []tenant.Workflow{}
	workflows = append(workflows, cfg.DefaultNamespace.Workflows...)
	for _, ns := range cfg.Namespaces {
		workflows = append(workflows, ns.Workflows...)
	}

	for _, h := range workflows {
		for _, step := range h.Steps {
			if step.IsFn() {
				modMap[step.ExecutableMod.FQMN] = true
			} else if step.IsGroup() {
				for _, mod := range step.Group {
					modMap[mod.FQMN] = true
				}
			}
		}
	}

	mods := []string{}
	for fn := range modMap {
		mods = append(mods, fn)
	}

	return mods
}

func DockerNameFromConfig(cfg *tenant.Config) (string, error) {
	identParts := strings.Split(cfg.Identifier, ".")
	if len(identParts) != 3 {
		return "", errors.New("ident has incorrect number of parts")
	}

	org := identParts[1]
	repo := identParts[2]

	name := fmt.Sprintf("%s/%s:%d", org, repo, cfg.TenantVersion)

	return name, nil
}
