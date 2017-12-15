package commands

import (
	"fmt"

	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	var purge bool
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var undeployCmd = &cobra.Command{
		Use:   "undeploy <DeploymentId>",
		Short: "Undeploy an application",
		Long:  `Undeploy an application specifying the deployment ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}

			url := "/deployments/" + args[0]
			if purge {
				url = url + "?purge=true"
			}

			request, err := client.NewRequest("DELETE", url, nil)
			if err != nil {
				errExit(err)
			}

			request.Header.Add("Accept", "application/json")
			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				errExit(err)
			}
			handleHTTPStatusCode(response, args[0], "deployment", http.StatusAccepted)

			fmt.Println("Undeployment submitted. In progress...")
			if shouldStreamLogs && !shouldStreamEvents {
				streamsLogs(client, args[0], !noColor, false, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				streamsEvents(client, args[0], !noColor, false, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}

			return nil
		},
	}

	deploymentsCmd.AddCommand(undeployCmd)
	undeployCmd.PersistentFlags().BoolVarP(&purge, "purge", "p", false, "To use if you want to purge instead of undeploy")
	undeployCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after undeploying the application. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	undeployCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after undeploying the CSAR.")

}
