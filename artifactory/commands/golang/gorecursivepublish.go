package golang

import (
	"github.com/jfrog/gocmd"
	"github.com/jfrog/gocmd/cmd"
	"github.com/jfrog/gocmd/executers"
	gocmdutils "github.com/jfrog/gocmd/executers/utils"
	"github.com/jfrog/gocmd/params"
	"github.com/jfrog/jfrog-cli-go/artifactory/utils"
	"github.com/jfrog/jfrog-cli-go/artifactory/utils/golang"
	"github.com/jfrog/jfrog-cli-go/utils/cliutils"
	"github.com/jfrog/jfrog-cli-go/utils/config"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
)

type GoRecursivePublishCommand struct {
	GoParamsCommand
}

func NewGoRecursivePublishCommand() *GoRecursivePublishCommand {
	return &GoRecursivePublishCommand{}
}

type GoParamsCommand struct {
	targetRepo string
	rtDetails  *config.ArtifactoryDetails
}

func (gpc *GoParamsCommand) RtDetails() (*config.ArtifactoryDetails, error) {
	return gpc.rtDetails, nil
}

func (gpc *GoParamsCommand) TargetRepo() string {
	return gpc.targetRepo
}

func (gpc *GoParamsCommand) SetTargetRepo(targetRepo string) *GoParamsCommand {
	gpc.targetRepo = targetRepo
	return gpc
}

func (gpc *GoParamsCommand) SetRtDetails(rtDetails *config.ArtifactoryDetails) *GoParamsCommand {
	gpc.rtDetails = rtDetails
	return gpc
}

func (gpc *GoParamsCommand) isRtDetailsEmpty() bool {
	if gpc.rtDetails != nil && reflect.DeepEqual(config.ArtifactoryDetails{}, gpc.rtDetails) {
		return false
	}
	return true
}

func (grp *GoRecursivePublishCommand) Run() error {
	rtDetails, err := grp.RtDetails()
	if errorutils.CheckError(err) != nil {
		return err
	}
	serviceManager, err := utils.CreateServiceManager(rtDetails, false)
	if err != nil {
		cliutils.ExitOnErr(err)
	}
	goModEditMessage := os.Getenv("JFROG_CLI_GO_MOD_EDIT_MSG")
	if goModEditMessage == "" {
		goModEditMessage = "// Generated by JFrog"
	} else {
		if !strings.HasPrefix(goModEditMessage, "//") {
			newContent := append([]byte("// "), goModEditMessage...)
			goModEditMessage = string(newContent)
		}
	}

	modFileExists, err := fileutils.IsFileExists("go.mod", false)
	if err != nil {
		return err
	}
	gmi := goModInfo{}
	wd, err := os.Getwd()
	if errorutils.CheckError(err) != nil {
		return gmi.revert(wd, err)
	}
	err = golang.LogGoVersion()
	if err != nil {
		return err
	}
	if !modFileExists {
		err = gmi.prepareModFile(wd, goModEditMessage)
		if err != nil {
			return err
		}
	} else {
		log.Debug("Using existing root mod file.")
		gmi.modFileContent, gmi.modFileStat, err = cmd.GetFileDetails("go.mod")
		if err != nil {
			return err
		}
	}

	goInfo := &params.ResolverDeployer{}
	deployerResolver := &params.Params{}
	deployerResolver.SetServiceManager(serviceManager).SetRepo(grp.TargetRepo())
	goInfo.SetDeployer(deployerResolver).SetResolver(deployerResolver)
	err = gocmd.RecursivePublish(goModEditMessage, goInfo)
	if errorutils.CheckError(err) != nil {
		if !modFileExists {
			log.Debug("Graph failed, preparing to run go mod tidy on the root project since got the following error:", err.Error())
			err = gmi.prepareAndRunTidyOnFailedGraph(wd, goModEditMessage, goInfo)
			if err != nil {
				return gmi.revert(wd, err)
			}
		} else {
			return gmi.revert(wd, err)
		}
	}

	err = os.Chdir(wd)
	if errorutils.CheckError(err) != nil {
		return gmi.revert(wd, err)
	}
	return gmi.revert(wd, nil)
}

func (grp *GoRecursivePublishCommand) CommandName() string {
	return "rt_go_recursive_publish"
}

type goModInfo struct {
	modFileContent      []byte
	modFileStat         os.FileInfo
	shouldRevertModFile bool
}

func (gmi *goModInfo) revert(wd string, err error) error {
	if gmi.shouldRevertModFile {
		log.Debug("Reverting to original go.mod of the root project")
		revertErr := ioutil.WriteFile("go.mod", gmi.modFileContent, gmi.modFileStat.Mode())
		if errorutils.CheckError(revertErr) != nil {
			if err != nil {
				log.Error(revertErr)
				return errorutils.CheckError(err)
			} else {
				return revertErr
			}
		}
	}
	return nil
}

func (gmi *goModInfo) prepareModFile(wd, goModEditMessage string) error {
	err := cmd.RunGoModInit("")
	if err != nil {
		return errorutils.CheckError(err)
	}
	regExp, err := gocmdutils.GetRegex()
	if err != nil {
		return errorutils.CheckError(err)
	}
	notEmptyModRegex := regExp.GetNotEmptyModRegex()
	gmi.modFileContent, gmi.modFileStat, err = cmd.GetFileDetails("go.mod")
	if err != nil {
		return err
	}
	projectPackage := executers.Package{}
	projectPackage.SetModContent(gmi.modFileContent)
	packageWithDep := executers.PackageWithDeps{Dependency: &projectPackage}
	if !packageWithDep.PatternMatched(notEmptyModRegex) {
		log.Debug("Root mod is empty, preparing to run 'go mod tidy'")
		err = cmd.RunGoModTidy()
		if errorutils.CheckError(err) != nil {
			return gmi.revert(wd, err)
		}
		gmi.shouldRevertModFile = true
	} else {
		log.Debug("Root project mod not empty.")
	}

	return nil
}

func (gmi *goModInfo) prepareAndRunTidyOnFailedGraph(wd, goModEditMessage string, goInfo *params.ResolverDeployer) error {
	// First revert the mod to an empty mod that includes only module name
	lines := strings.Split(string(gmi.modFileContent), "\n")
	emptyMod := strings.Join(lines[:3], "\n")
	gmi.modFileContent = []byte(emptyMod)
	gmi.shouldRevertModFile = true
	err := gmi.revert(wd, nil)
	if err != nil {
		log.Error(err)
	}
	// Run go mod tidy.
	err = cmd.RunGoModTidy()
	if err != nil {
		return errorutils.CheckError(err)
	}
	// Perform collection again after tidy finished successfully.
	err = gocmd.RecursivePublish(goModEditMessage, goInfo)
	if errorutils.CheckError(err) != nil {
		return gmi.revert(wd, err)
	}
	return nil
}
