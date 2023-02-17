package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/util"
)

func StatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Get K8s deployment status",
		Long:  "Get Kubernetes pods and service status for your E2 Core deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			util.LogInfo("Pods:")
			util.Command.Run("kubectl get pods -n suborbital")

			fmt.Println()
			util.LogInfo("Services:")
			util.Command.Run("kubectl get svc -n suborbital")

			return nil
		},
	}
}
