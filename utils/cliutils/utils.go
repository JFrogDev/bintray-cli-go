package cliutils

import (
	"github.com/jfrog/jfrog-cli-core/utils/coreutils"
	"os"
	"strings"

	serviceutils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/io/content"

	"github.com/codegangsta/cli"
	"github.com/jfrog/jfrog-cli/utils/summary"
	"github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/pkg/errors"

	clientutils "github.com/jfrog/jfrog-client-go/utils"
)

// Error modes (how should the application behave when the CheckError function is invoked):
type OnError string

var cliTempDir string

func init() {
	// Initialize error handling.
	if os.Getenv(ErrorHandling) == string(OnErrorPanic) {
		errorutils.CheckError = PanicOnError
	}

	// Initialize the temp base-dir path of the CLI executions.
	cliTempDir = os.Getenv(TempDir)
	if cliTempDir == "" {
		cliTempDir = os.TempDir()
	}
	fileutils.SetTempDirBase(cliTempDir)

	// Initialize cli-core values.
	cliUserAgent := os.Getenv(UserAgent)
	if cliUserAgent == "" {
		cliUserAgent = ClientAgent + "/" + CliVersion
	}
	coreutils.SetCliUserAgent(cliUserAgent)
	coreutils.SetClientAgent(ClientAgent)
	coreutils.SetVersion(CliVersion)
}

func PanicOnError(err error) error {
	if err != nil {
		panic(err)
	}
	return err
}

func ExitOnErr(err error) {
	if err, ok := err.(coreutils.CliError); ok {
		traceExit(err.ExitCode, err)
	}
	if exitCode := coreutils.GetExitCode(err, 0, 0, false); exitCode != coreutils.ExitCodeNoError {
		traceExit(exitCode, err)
	}
}

func GetCliError(err error, success, failed int, failNoOp bool) error {
	switch coreutils.GetExitCode(err, success, failed, failNoOp) {
	case coreutils.ExitCodeError:
		{
			var errorMessage string
			if err != nil {
				errorMessage = err.Error()
			}
			return coreutils.CliError{ExitCode: coreutils.ExitCodeError, ErrorMsg: errorMessage}
		}
	case coreutils.ExitCodeFailNoOp:
		return coreutils.CliError{ExitCode: coreutils.ExitCodeFailNoOp, ErrorMsg: "No errors, but also no files affected (fail-no-op flag)."}
	default:
		return nil
	}
}

func traceExit(exitCode coreutils.ExitCode, err error) {
	if err != nil && len(err.Error()) > 0 {
		log.Error(err)
	}
	os.Exit(exitCode.Code)
}

type detailedSummaryRecord struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Print summary report.
// a given non-nil error will pass through and be returned as is if no other errors are raised.
// In case of a nil error, the current function error will be returned.
func summaryPrintError(summaryError, originalError error) error {
	if originalError != nil {
		if summaryError != nil {
			log.Error(summaryError)
		}
		return originalError
	}
	return summaryError
}

// If a resultReader is provided, we will iterate over the result and print a detailed summary including the affected files.
func PrintSummaryReport(success, failed int, reader *content.ContentReader, rtUrl string, originalErr error) error {
	basicSummary, mErr := CreateSummaryReportString(success, failed, originalErr)
	if mErr != nil {
		return summaryPrintError(mErr, originalErr)
	}
	// A reader wasn't provided, prints the basic summary json and return.
	if reader == nil {
		log.Output(basicSummary)
		return summaryPrintError(mErr, originalErr)
	}
	reader.Reset()
	defer reader.Close()
	writer, mErr := content.NewContentWriter("files", false, true)
	if mErr != nil {
		log.Output(basicSummary)
		return summaryPrintError(mErr, originalErr)
	}
	// We remove the closing curly bracket in order to append the affected files array using a responseWriter to write directly to stdout.
	basicSummary = strings.TrimSuffix(basicSummary, "\n}") + ","
	log.Output(basicSummary)
	defer log.Output("}")
	for file := new(serviceutils.FileInfo); reader.NextRecord(file) == nil; file = new(serviceutils.FileInfo) {
		record := detailedSummaryRecord{
			Source: rtUrl + file.ArtifactoryPath,
			Target: file.LocalPath,
		}
		writer.Write(record)
	}
	mErr = writer.Close()
	return summaryPrintError(mErr, originalErr)
}

func CreateSummaryReportString(success, failed int, err error) (string, error) {
	summaryReport := summary.New(err)
	summaryReport.Totals.Success = success
	summaryReport.Totals.Failure = failed
	if err == nil && summaryReport.Totals.Failure != 0 {
		summaryReport.Status = summary.Failure
	}
	content, mErr := summaryReport.Marshal()
	if errorutils.CheckError(mErr) != nil {
		return "", mErr
	}
	return utils.IndentJson(content), mErr
}

func PrintHelpAndReturnError(msg string, context *cli.Context) error {
	log.Error(msg + " " + GetDocumentationMessage())
	cli.ShowCommandHelp(context, context.Command.Name)
	return errors.New(msg)
}

// This function indicates whether the command should be executed without
// confirmation warning or not.
// If the --quiet option was sent, it is used to determine whether to prompt the confirmation or not.
// If not, the command will prompt the confirmation, unless the CI environment variable was set to true.
func GetQuietValue(c *cli.Context) bool {
	if c.IsSet("quiet") {
		return c.Bool("quiet")
	}

	return getCiValue()
}

// This function indicates whether the command should be executed in
// an interactive mode.
// If the --interactive option was sent, it is used to determine the mode.
// If not, the mode will be interactive, unless the CI environment variable was set to true.
func GetInteractiveValue(c *cli.Context) bool {
	if c.IsSet("interactive") {
		return c.BoolT("interactive")
	}

	return !getCiValue()
}

// Return true if the CI environment variable was set to true.
func getCiValue() bool {
	var ci bool
	var err error
	if ci, err = clientutils.GetBoolEnvValue(coreutils.CI, false); err != nil {
		return false
	}
	return ci
}

func GetVersion() string {
	return CliVersion
}

func GetDocumentationMessage() string {
	return "You can read the documentation at https://www.jfrog.com/confluence/display/CLI/JFrog+CLI"
}

func GetBuildName(buildName string) string {
	return getOrDefaultEnv(buildName, coreutils.BuildName)
}

func GetBuildUrl(buildUrl string) string {
	return getOrDefaultEnv(buildUrl, BuildUrl)
}

func GetEnvExclude(envExclude string) string {
	return getOrDefaultEnv(envExclude, EnvExclude)
}

// Return argument if not empty or retrieve from environment variable
func getOrDefaultEnv(arg, envKey string) string {
	if arg != "" {
		return arg
	}
	return os.Getenv(envKey)
}
