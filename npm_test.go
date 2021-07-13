package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/common/spec"
	clientutils "github.com/jfrog/jfrog-client-go/utils"

	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/npm"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/ioutils"
	"github.com/jfrog/jfrog-cli/inttestutils"
	"github.com/jfrog/jfrog-cli/utils/tests"
	"github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/stretchr/testify/assert"
)

type npmTestParams struct {
	command        string
	repo           string
	npmArgs        string
	wd             string
	buildNumber    string
	moduleName     string
	validationFunc func(*testing.T, npmTestParams)
}

func cleanNpmTest() {
	os.Unsetenv(coreutils.HomeDir)
	deleteSpec := spec.NewBuilder().Pattern(tests.NpmRepo).BuildSpec()
	tests.DeleteFiles(deleteSpec, serverDetails)
	tests.CleanFileSystem()
}

func TestNpm(t *testing.T) {
	initNpmTest(t)
	defer cleanNpmTest()
	wd, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(wd)

	npmProjectPath, npmScopedProjectPath, npmNpmrcProjectPath, npmProjectCi := initNpmFilesTest(t)
	var npmTests = []npmTestParams{
		{command: "npmci", repo: tests.NpmRemoteRepo, wd: npmProjectCi, validationFunc: validateNpmInstall},
		{command: "npmci", repo: tests.NpmRemoteRepo, wd: npmProjectCi, moduleName: ModuleNameJFrogTest, validationFunc: validateNpmInstall},
		{command: "npm-install", repo: tests.NpmRemoteRepo, wd: npmProjectPath, moduleName: ModuleNameJFrogTest, validationFunc: validateNpmInstall},
		{command: "npm-install", repo: tests.NpmRemoteRepo, wd: npmScopedProjectPath, validationFunc: validateNpmInstall},
		{command: "npm-install", repo: tests.NpmRemoteRepo, wd: npmNpmrcProjectPath, validationFunc: validateNpmInstall},
		{command: "npm-install", repo: tests.NpmRemoteRepo, wd: npmProjectPath, validationFunc: validateNpmInstall, npmArgs: "--production"},
		{command: "npmi", repo: tests.NpmRemoteRepo, wd: npmNpmrcProjectPath, validationFunc: validateNpmPackInstall, npmArgs: "yaml"},
		{command: "npmp", repo: tests.NpmRepo, wd: npmScopedProjectPath, moduleName: ModuleNameJFrogTest, validationFunc: validateNpmScopedPublish},
		{command: "npm-publish", repo: tests.NpmRepo, wd: npmProjectPath, validationFunc: validateNpmPublish},
	}

	for i, npmTest := range npmTests {
		err = os.Chdir(filepath.Dir(npmTest.wd))
		assert.NoError(t, err)
		npmrcFileInfo, err := os.Stat(".npmrc")
		if err != nil && !os.IsNotExist(err) {
			assert.Fail(t, err.Error())
		}
		var buildNumber string
		commandArgs := []string{npmTest.command}
		buildNumber = strconv.Itoa(i + 100)
		commandArgs = append(commandArgs, npmTest.npmArgs)
		commandArgs = append(commandArgs, "--build-name="+tests.NpmBuildName, "--build-number="+buildNumber)

		if npmTest.moduleName != "" {
			runNpm(t, append(commandArgs, "--module="+npmTest.moduleName)...)
		} else {
			npmTest.moduleName = readModuleId(t, npmTest.wd)
			runNpm(t, commandArgs...)
		}
		validatePartialsBuildInfo(t, tests.NpmBuildName, buildNumber, npmTest.moduleName)
		artifactoryCli.Exec("bp", tests.NpmBuildName, buildNumber)
		npmTest.buildNumber = buildNumber
		npmTest.validationFunc(t, npmTest)

		// make sure npmrc file was not changed (if existed)
		postTestFileInfo, postTestFileInfoErr := os.Stat(".npmrc")
		validateNpmrcFileInfo(t, npmTest, npmrcFileInfo, postTestFileInfo, err, postTestFileInfoErr)
	}

	inttestutils.DeleteBuild(serverDetails.ArtifactoryUrl, tests.NpmBuildName, artHttpDetails)
}

func readModuleId(t *testing.T, wd string) string {
	packageInfo, err := coreutils.ReadPackageInfoFromPackageJson(filepath.Dir(wd))
	assert.NoError(t, err)
	return packageInfo.BuildInfoModuleId()
}

func TestNpmWithGlobalConfig(t *testing.T) {
	initNpmTest(t)
	defer cleanNpmTest()
	wd, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(wd)
	npmProjectPath := initGlobalNpmFilesTest(t)
	err = os.Chdir(filepath.Dir(npmProjectPath))
	assert.NoError(t, err)
	runNpm(t, "npm-install", "--build-name="+tests.NpmBuildName, "--build-number=1", "--module="+ModuleNameJFrogTest)
	validatePartialsBuildInfo(t, tests.NpmBuildName, "1", ModuleNameJFrogTest)
}

func validatePartialsBuildInfo(t *testing.T, buildName, buildNumber, moduleName string) {
	partials, err := utils.ReadPartialBuildInfoFiles(buildName, buildNumber, "")
	assert.NoError(t, err)
	for _, module := range partials {
		assert.Equal(t, moduleName, module.ModuleId)
		assert.Equal(t, buildinfo.Npm, module.ModuleType)
		assert.NotZero(t, module.Timestamp)
	}
}

func validateNpmrcFileInfo(t *testing.T, npmTest npmTestParams, npmrcFileInfo, postTestNpmrcFileInfo os.FileInfo, err, postTestFileInfoErr error) {
	if postTestFileInfoErr != nil && !os.IsNotExist(postTestFileInfoErr) {
		assert.Fail(t, postTestFileInfoErr.Error())
	}
	assert.False(t, err == nil && postTestFileInfoErr != nil, ".npmrc file existed and was not restored at the end of the install command.")
	assert.False(t, err != nil && postTestFileInfoErr == nil, ".npmrc file was not deleted at the end of the install command.")
	assert.False(t, err == nil && postTestFileInfoErr == nil && (npmrcFileInfo.Mode() != postTestNpmrcFileInfo.Mode() || npmrcFileInfo.Size() != postTestNpmrcFileInfo.Size()),
		".npmrc file was changed after running npm command! it was:\n%v\nnow it is:\n%v\nTest arguments are:\n%v", npmrcFileInfo, postTestNpmrcFileInfo, npmTest)
	// make sue the temp .npmrc was deleted.
	bcpNpmrc, err := os.Stat("jfrog.npmrc.backup")
	if err != nil && !os.IsNotExist(err) {
		assert.Fail(t, err.Error())
	}
	assert.Nil(t, bcpNpmrc, "The file 'jfrog.npmrc.backup' was supposed to be deleted but it was not when running the configuration:\n%v", npmTest)
}

func initNpmFilesTest(t *testing.T) (npmProjectPath, npmScopedProjectPath, npmNpmrcProjectPath, npmProjectCi string) {
	npmProjectPath, err := filepath.Abs(createNpmProject(t, "npmproject"))
	assert.NoError(t, err)
	npmScopedProjectPath, err = filepath.Abs(createNpmProject(t, "npmscopedproject"))
	assert.NoError(t, err)
	npmNpmrcProjectPath, err = filepath.Abs(createNpmProject(t, "npmnpmrcproject"))
	assert.NoError(t, err)
	npmProjectCi, err = filepath.Abs(createNpmProject(t, "npmprojectci"))
	assert.NoError(t, err)
	prepareArtifactoryForNpmBuild(t, filepath.Dir(npmProjectPath))
	prepareArtifactoryForNpmBuild(t, filepath.Dir(npmProjectCi))
	err = createConfigFileForTest([]string{filepath.Dir(npmProjectPath), filepath.Dir(npmScopedProjectPath),
		filepath.Dir(npmNpmrcProjectPath), filepath.Dir(npmProjectCi)}, tests.NpmRemoteRepo, tests.NpmRepo, t, utils.Npm, false)
	assert.NoError(t, err)
	return
}

func initNpmProjectTest(t *testing.T) (npmProjectPath string) {
	npmProjectPath, err := filepath.Abs(createNpmProject(t, "npmproject"))
	assert.NoError(t, err)
	prepareArtifactoryForNpmBuild(t, filepath.Dir(npmProjectPath))
	err = createConfigFileForTest([]string{filepath.Dir(npmProjectPath)}, tests.NpmRemoteRepo, tests.NpmRepo, t, utils.Npm, false)
	assert.NoError(t, err)
	return
}

func initGlobalNpmFilesTest(t *testing.T) (npmProjectPath string) {
	npmProjectPath, err := filepath.Abs(createNpmProject(t, "npmproject"))
	assert.NoError(t, err)

	prepareArtifactoryForNpmBuild(t, filepath.Dir(npmProjectPath))
	jfrogHomeDir, err := coreutils.GetJfrogHomeDir()
	assert.NoError(t, err)
	err = createConfigFileForTest([]string{jfrogHomeDir}, tests.NpmRemoteRepo, tests.NpmRepo, t, utils.Npm, true)
	assert.NoError(t, err)

	return
}

func createNpmProject(t *testing.T, dir string) string {
	srcPackageJson := filepath.Join(filepath.FromSlash(tests.GetTestResourcesPath()), "npm", dir, "package.json")
	targetPackageJson := filepath.Join(tests.Out, dir)
	packageJson, err := tests.ReplaceTemplateVariables(srcPackageJson, targetPackageJson)
	assert.NoError(t, err)

	// failure can be ignored
	npmrcExists, err := fileutils.IsFileExists(filepath.Join(filepath.Dir(srcPackageJson), ".npmrc"), false)
	assert.NoError(t, err)

	if npmrcExists {
		_, err = tests.ReplaceTemplateVariables(filepath.Join(filepath.Dir(srcPackageJson), ".npmrc"), targetPackageJson)
		assert.NoError(t, err)
	}
	return packageJson
}

func validateNpmInstall(t *testing.T, npmTestParams npmTestParams) {
	expectedDependencies := []expectedDependency{{id: "xml:1.0.1", scopes: []string{"prod"}}}
	if !strings.Contains(npmTestParams.npmArgs, "-only=prod") && !strings.Contains(npmTestParams.npmArgs, "-production") {
		expectedDependencies = append(expectedDependencies, expectedDependency{id: "json:9.0.6", scopes: []string{"dev"}})
	}
	publishedBuildInfo, found, err := tests.GetBuildInfo(serverDetails, tests.NpmBuildName, npmTestParams.buildNumber)
	if err != nil {
		assert.NoError(t, err)
		return
	}
	if !found {
		assert.True(t, found, "build info was expected to be found")
		return
	}
	buildInfo := publishedBuildInfo.BuildInfo
	if buildInfo.Modules == nil || len(buildInfo.Modules) == 0 {
		// Case no module was created
		t.Error(fmt.Sprintf("npm install test with command '%s' and repo '%s' failed", npmTestParams.command, npmTestParams.repo))
		return
	}

	equalDependenciesSlices(t, expectedDependencies, buildInfo.Modules[0].Dependencies)
}

type expectedDependency struct {
	id     string
	scopes []string
}

func validateNpmPackInstall(t *testing.T, npmTestParams npmTestParams) {
	publishedBuildInfo, found, err := tests.GetBuildInfo(serverDetails, tests.NpmBuildName, npmTestParams.buildNumber)
	if err != nil {
		assert.NoError(t, err)
		return
	}
	if !found {
		assert.True(t, found, "build info was expected to be found")
		return
	}
	buildInfo := publishedBuildInfo.BuildInfo
	assert.Zero(t, buildInfo.Modules, "npm install test with the arguments: \n%v \nexpected to have no modules")

	packageJsonFile, err := ioutil.ReadFile(npmTestParams.wd)
	assert.NoError(t, err)

	var packageJson struct {
		Dependencies map[string]string `json:"dependencies,omitempty"`
	}
	assert.NoError(t, json.Unmarshal(packageJsonFile, &packageJson))
	assert.False(t, len(packageJson.Dependencies) != 2 || packageJson.Dependencies[npmTestParams.npmArgs] == "",
		"npm install test with the arguments: \n%v \nexpected have the dependency %v in the following package.json file: \n%v",
		npmTestParams, npmTestParams.npmArgs, packageJsonFile)
}

func validateNpmPublish(t *testing.T, npmTestParams npmTestParams) {
	verifyExistInArtifactoryByProps(tests.GetNpmDeployedArtifacts(),
		tests.NpmRepo+"/*",
		fmt.Sprintf("build.name=%v;build.number=%v;build.timestamp=*", tests.NpmBuildName, npmTestParams.buildNumber), t)
	validateNpmCommonPublish(t, npmTestParams)
}

func validateNpmScopedPublish(t *testing.T, npmTestParams npmTestParams) {
	verifyExistInArtifactoryByProps(tests.GetNpmDeployedScopedArtifacts(),
		tests.NpmRepo+"/*",
		fmt.Sprintf("build.name=%v;build.number=%v;build.timestamp=*", tests.NpmBuildName, npmTestParams.buildNumber), t)
	validateNpmCommonPublish(t, npmTestParams)
}

func validateNpmCommonPublish(t *testing.T, npmTestParams npmTestParams) {
	publishedBuildInfo, found, err := tests.GetBuildInfo(serverDetails, tests.NpmBuildName, npmTestParams.buildNumber)
	if err != nil {
		assert.NoError(t, err)
		return
	}
	if !found {
		assert.True(t, found, "build info was expected to be found")
		return
	}
	buildInfo := publishedBuildInfo.BuildInfo
	expectedArtifactName := "jfrog-cli-tests-1.0.0.tgz"
	if buildInfo.Modules == nil || len(buildInfo.Modules) == 0 {
		// Case no module was created
		assert.Fail(t, "npm publish test with the arguments: \n%v \nexpected to have module with the following artifact: \n%v \nbut has no modules: \n%v",
			npmTestParams, expectedArtifactName, buildInfo)
		return
	}
	// The checksums are ignored when comparing the actual and the expected
	assert.Len(t, buildInfo.Modules[0].Artifacts, 1, "npm publish test with the arguments: \n%v \nexpected to have the following artifact: \n%v \nbut has: \n%v",
		npmTestParams, expectedArtifactName, buildInfo.Modules[0].Artifacts)
	assert.Equal(t, npmTestParams.moduleName, buildInfo.Modules[0].Id, "npm publish test with the arguments: \n%v \nexpected to have the following module name: \n%v \nbut has: \n%v",
		npmTestParams, npmTestParams.moduleName, buildInfo.Modules[0].Id)
	assert.Equal(t, expectedArtifactName, buildInfo.Modules[0].Artifacts[0].Name, "npm publish test with the arguments: \n%v \nexpected to have the following artifact: \n%v \nbut has: \n%v",
		npmTestParams, expectedArtifactName, buildInfo.Modules[0].Artifacts[0].Name)
}

func prepareArtifactoryForNpmBuild(t *testing.T, workingDirectory string) {
	assert.NoError(t, os.Chdir(workingDirectory))

	caches := ioutils.DoubleWinPathSeparator(filepath.Join(workingDirectory, "caches"))
	// Run install with -cache argument to download the artifacts from Artifactory
	// This done to be sure the artifacts exists in Artifactory
	artifactoryCli.Exec("npm-install", tests.NpmRemoteRepo, "--npm-args=-cache="+caches)

	assert.NoError(t, os.RemoveAll(filepath.Join(workingDirectory, "node_modules")))
	assert.NoError(t, os.RemoveAll(caches))
}

func initNpmTest(t *testing.T) {
	if !*tests.TestNpm {
		t.Skip("Skipping Npm test. To run Npm test add the '-test.npm=true' option.")
	}
	createJfrogHomeConfig(t, true)
}

func runNpm(t *testing.T, args ...string) {
	var err error
	err = artifactoryCli.WithoutCredentials().Exec(args...)
	assert.NoError(t, err)
}

func TestNpmPublishDetailedSummary(t *testing.T) {
	initNpmTest(t)
	defer cleanNpmTest()
	wd, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(wd)
	// Init npm project & npmp command for testing
	npmProjectPath := strings.TrimSuffix(initNpmProjectTest(t), "package.json")
	configFilePath := filepath.Join(npmProjectPath, ".jfrog", "projects", "npm.yaml")
	args := []string{"--detailed-summary=true"}
	npmpCmd := npm.NewNpmPublishCommand()
	npmpCmd.SetConfigFilePath(configFilePath).SetArgs(args)

	err = commands.Exec(npmpCmd)
	assert.NoError(t, err)

	result := npmpCmd.Result()
	assert.NotNil(t, result)
	reader := result.Reader()
	assert.NoError(t, reader.GetError())
	defer reader.Close()
	// Read result
	var files []clientutils.FileTransferDetails
	for transferDetails := new(clientutils.FileTransferDetails); reader.NextRecord(transferDetails) == nil; transferDetails = new(clientutils.FileTransferDetails) {
		files = append(files, *transferDetails)
	}
	// Verify deploy details
	expectedSourcePath := npmProjectPath + "jfrog-cli-tests-v1.0.0.tgz"
	expectedTargetPath := serverDetails.ArtifactoryUrl + tests.NpmRepo + "/jfrog-cli-tests/-/jfrog-cli-tests-1.0.0.tgz"
	assert.Equal(t, expectedSourcePath, files[0].SourcePath, "Summary validation failed - unmatched SourcePath.")
	assert.Equal(t, expectedTargetPath, files[0].TargetPath, "Summary validation failed - unmatched TargetPath.")
	assert.Equal(t, 1, len(files), "Summary validation failed - only one archive should be deployed.")
	// Verify sha256 is valid (a string size 256 characters) and not an empty string.
	assert.Equal(t, 64, len(files[0].Sha256), "Summary validation failed - sha256 should be in size 64 digits.")
}

func TestYarn(t *testing.T) {
	initNpmTest(t)

	testDataSource := filepath.Join(filepath.FromSlash(tests.GetTestResourcesPath()), "yarn")
	testDataTarget := filepath.Join(tests.Out, "yarn")
	err := fileutils.CopyDir(testDataSource, testDataTarget, true, nil)
	assert.NoError(t, err)
	defer cleanNpmTest()

	yarnProjectPath := filepath.Join(testDataTarget, "yarnproject")
	err = createConfigFileForTest([]string{yarnProjectPath}, tests.NpmRemoteRepo, "", t, utils.Yarn, false)
	assert.NoError(t, err)
	wd, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(wd)
	err = os.Chdir(yarnProjectPath)
	assert.NoError(t, err)

	err = artifactoryCli.WithoutCredentials().Exec("yarn", "--build-name="+tests.YarnBuildName, "--build-number=1", "--module="+ModuleNameJFrogTest)
	assert.NoError(t, err)

	validatePartialsBuildInfo(t, tests.YarnBuildName, "1", ModuleNameJFrogTest)

	err = artifactoryCli.WithoutCredentials().Exec("bp", tests.YarnBuildName, "1")
	assert.NoError(t, err)
	publishedBuildInfo, found, err := tests.GetBuildInfo(serverDetails, tests.YarnBuildName, "1")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 1, len(publishedBuildInfo.BuildInfo.Modules))
	assert.Equal(t, buildinfo.Npm, publishedBuildInfo.BuildInfo.Modules[0].Type)
	assert.Equal(t, "jfrog-test", publishedBuildInfo.BuildInfo.Modules[0].Id)
	assert.Equal(t, 0, len(publishedBuildInfo.BuildInfo.Modules[0].Artifacts))

	expectedDependencies := []expectedDependency{{id: "xml:1.0.1"}, {id: "json:9.0.6"}}
	equalDependenciesSlices(t, expectedDependencies, publishedBuildInfo.BuildInfo.Modules[0].Dependencies)

	inttestutils.DeleteBuild(serverDetails.ArtifactoryUrl, tests.YarnBuildName, artHttpDetails)
}

// Checks if the expected dependencies match the actual dependencies. Only the dependencies' IDs and scopes (not more than one scope) are compared.
func equalDependenciesSlices(t *testing.T, expectedDependencies []expectedDependency, actualDependencies []buildinfo.Dependency) {
	assert.Equal(t, len(expectedDependencies), len(actualDependencies))
	for _, dependency := range expectedDependencies {
		found := false
		for _, actualDependency := range actualDependencies {
			if actualDependency.Id == dependency.id &&
				len(actualDependency.Scopes) == len(dependency.scopes) &&
				(len(actualDependency.Scopes) == 0 || actualDependency.Scopes[0] == dependency.scopes[0]) {
				found = true
				break
			}
		}
		// The checksums are ignored when comparing the actual and the expected
		assert.True(t, found, "The dependencies from the build-info did not match the expected. expected: %v, actual: %v",
			expectedDependencies, dependenciesToPrintableArray(actualDependencies))
	}
}
