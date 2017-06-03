package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/mesosphere/dcos-commons/cli/client"
	"github.com/mesosphere/dcos-commons/cli/config"
	"gopkg.in/alecthomas/kingpin.v2"
)

type DescribeHandler struct{}

type DescribeRequest struct {
	AppID string `json:"appId"`
}

func reportErrorAndExit(err error, responseBytes []byte) {
	client.LogMessage("Failed to unmarshal response. Error: %s", err)
	client.LogMessage("Original data follows:")
	client.LogMessageAndExit(string(responseBytes))
}

func doDescribe() {
	requestContent, _ := json.Marshal(DescribeRequest{config.ServiceName})
	response := client.HTTPCosmosPostJSON("describe", string(requestContent))
	responseBytes := client.GetResponseBytes(response)
	// This attempts to retrieve resolvedOptions from the response. This field is only provided by
	// Cosmos running on Enterprise DC/OS 1.10 clusters or later.
	resolvedOptionsBytes, err := client.GetValueFromJSONResponse(responseBytes, "resolvedOptions")
	if err != nil {
		reportErrorAndExit(err, responseBytes)
	}
	if resolvedOptionsBytes != nil {
		client.PrintJSONBytes(resolvedOptionsBytes, nil)
	} else {
		client.LogMessage("Package configuration is not available for service %s.", config.ServiceName)
		client.LogMessageAndExit("Only packages installed with Enterprise DC/OS 1.10 or newer will have configuration persisted.")
	}
}

func (cmd *DescribeHandler) DescribeConfiguration(c *kingpin.ParseContext) error {
	config.Command = c.SelectedCommand.FullCommand()
	doDescribe()
	return nil
}

func HandleDescribe(app *kingpin.Application) {
	cmd := &DescribeHandler{}
	app.Command("describe", "View the package configuration for this DC/OS service").Action(cmd.DescribeConfiguration)
}

type UpdateHandler struct {
	UpdateName     string
	OptionsFile    string
	PackageVersion string
	ViewStatus     bool
}

type UpdateRequest struct {
	AppID          string                 `json:"appId"`
	PackageVersion string                 `json:"packageVersion,omitempty"`
	OptionsJSON    map[string]interface{} `json:"options,omitempty"`
}

func printPackageVersions() {
	requestContent, _ := json.Marshal(DescribeRequest{config.ServiceName})
	response := client.HTTPCosmosPostJSON("describe", string(requestContent))
	responseBytes := client.GetResponseBytes(response)
	packageBytes, err := client.GetValueFromJSONResponse(responseBytes, "package")
	if err != nil {
		reportErrorAndExit(err, responseBytes)
	}
	currentVersionBytes, err := client.GetValueFromJSONResponse(packageBytes, "version")
	if err != nil {
		reportErrorAndExit(err, responseBytes)
	}
	downgradeVersionsBytes, err := client.GetValueFromJSONResponse(responseBytes, "downgradesTo")
	if err != nil {
		reportErrorAndExit(err, responseBytes)
	}
	upgradeVersionsBytes, err := client.GetValueFromJSONResponse(responseBytes, "upgradesTo")
	if err != nil {
		reportErrorAndExit(err, responseBytes)
	}
	client.LogMessage("Current package version is: %s", currentVersionBytes)
	if downgradeVersionsBytes != nil {
		client.LogMessage("Valid package downgrade versions: %s", downgradeVersionsBytes)
	}
	if upgradeVersionsBytes != nil {
		client.LogMessage("Valid package upgrade versions: %s", upgradeVersionsBytes)
	}

}

func (cmd *UpdateHandler) ViewPackageVersions(c *kingpin.ParseContext) error {
	config.Command = c.SelectedCommand.FullCommand()
	printPackageVersions()
	return nil
}

func readJSONFile(filename string) (map[string]interface{}, error) {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return client.UnmarshalJSON(fileBytes)
}

func parseUpdateResponse(responseBytes []byte) (string, error) {
	responseJSON, err := client.UnmarshalJSON(responseBytes)
	if err != nil {
		return "", err
	}
	return string(responseJSON["marathonDeploymentId"].(string)), nil
}

func doUpdate(optionsFile, packageVersion string) {
	request := UpdateRequest{AppID: config.ServiceName}
	if len(packageVersion) == 0 && len(optionsFile) == 0 {
		client.LogMessageAndExit("Either --options and/or --package-version must be specified. See --help.")
	}
	if len(packageVersion) > 0 {
		request.PackageVersion = packageVersion
	}
	if len(optionsFile) > 0 {
		fileBytes, err := ioutil.ReadFile(optionsFile)
		if err != nil {
			client.LogMessageAndExit("Failed to load specified options file %s: %s", optionsFile, err)
			return
		}
		optionsJSON, err := client.UnmarshalJSON(fileBytes)
		if err != nil {
			client.LogMessageAndExit("Failed to parse JSON in specified options file %s: %s", optionsFile, err)
			return
		}
		request.OptionsJSON = optionsJSON
	}
	requestContent, _ := json.Marshal(request)
	response := client.HTTPCosmosPostJSON("update", string(requestContent))
	responseBytes := client.GetResponseBytes(response)
	_, err := client.UnmarshalJSON(responseBytes)
	if err != nil {
		reportErrorAndExit(err, responseBytes)
	}
	client.LogMessage(fmt.Sprintf("Update started. Please use `dcos %s --name=%s update status` to view progress.", config.ModuleName, config.ServiceName))
}

func (cmd *UpdateHandler) UpdateConfiguration(c *kingpin.ParseContext) error {
	config.Command = c.SelectedCommand.FullCommand()
	doUpdate(cmd.OptionsFile, cmd.PackageVersion)
	return nil
}

func HandleUpdateSection(app *kingpin.Application) {
	cmd := &UpdateHandler{}
	update := app.Command("update", "Updates the package version or configuration for this DC/OS service")

	start := update.Command("start", "Launches an update operation").Action(cmd.UpdateConfiguration)
	start.Flag("options", "Path to a JSON file that contains customized package installation options").StringVar(&cmd.OptionsFile)
	start.Flag("package-version", "The desired package version").StringVar(&cmd.PackageVersion)

	planCmd := &PlanHandler{}

	forceComplete := update.Command("force-complete", "Force complete a specific step in the provided phase").Alias("force").Action(planCmd.RunForceComplete)
	// TODO: it'd be nice if this was optional (but there is no way to make this optional and have required arguments after it).
	forceComplete.Arg("plan", "Name of the plan to force complete").Required().StringVar(&planCmd.PlanName)
	forceComplete.Arg("phase", "Name or UUID of the phase containing the provided step").Required().StringVar(&planCmd.Phase)
	forceComplete.Arg("step", "Name or UUID of step to be restarted").Required().StringVar(&planCmd.Step)

	forceRestart := update.Command("force-restart", "Restart a deploy plan, or specific step in the provided phase").Alias("restart").Action(planCmd.RunForceRestart)
	forceRestart.Arg("plan", "Name of the plan to restart").StringVar(&planCmd.PlanName)
	forceRestart.Arg("phase", "Name or UUID of the phase containing the provided step").StringVar(&planCmd.Phase)
	forceRestart.Arg("step", "Name or UUID of step to be restarted").StringVar(&planCmd.Step)

	update.Command("package-versions", "View a list of available package versions to downgrade or upgrade to").Action(cmd.ViewPackageVersions)

	pause := update.Command("pause", "Pause the deploy plan, or the plan with the provided name, or a specific phase in that plan with the provided name or UUID").Alias("interrupt").Action(planCmd.RunPause)
	pause.Arg("plan", "Name of the plan to pause").StringVar(&planCmd.PlanName)

	resume := update.Command("resume", "Resume the deploy plan, or the plan with the provided name, or a specific phase in that plan with the provided name or UUID").Alias("continue").Action(planCmd.RunResume)
	resume.Arg("plan", "Name of the plan to resume").StringVar(&planCmd.PlanName)

	status := update.Command("status", "View status of a running update").Alias("show").Action(planCmd.RunStatus)
	status.Arg("plan", "Name of the plan to show").StringVar(&planCmd.PlanName)
	status.Flag("json", "Show raw JSON response instead of user-friendly tree").BoolVar(&planCmd.RawJSON)
}
