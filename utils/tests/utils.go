package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/jfrog/jfrog-cli/utils/summary"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/jfrog/jfrog-client-go/artifactory/services"

	corelog "github.com/jfrog/jfrog-cli-core/utils/log"

	"github.com/jfrog/jfrog-cli-core/artifactory/commands/generic"
	"github.com/jfrog/jfrog-cli-core/artifactory/spec"
	artUtils "github.com/jfrog/jfrog-cli-core/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/utils/config"
	"github.com/jfrog/jfrog-cli-core/utils/coreutils"
	"github.com/stretchr/testify/assert"

	commandutils "github.com/jfrog/jfrog-cli-core/artifactory/commands/utils"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/auth"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

var RtUrl *string
var RtUser *string
var RtPassword *string
var RtApiKey *string
var RtSshKeyPath *string
var RtSshPassphrase *string
var RtAccessToken *string
var RtDistributionUrl *string
var RtDistributionAccessToken *string
var BtUser *string
var BtKey *string
var BtOrg *string
var TestArtifactory *bool
var TestBintray *bool
var TestArtifactoryProxy *bool
var TestDistribution *bool
var TestDocker *bool
var TestGo *bool
var TestNpm *bool
var TestGradle *bool
var TestMaven *bool
var DockerRepoDomain *string
var DockerVirtualRepo *string
var DockerRemoteRepo *string
var DockerLocalRepo *string
var TestNuget *bool
var HideUnitTestLog *bool
var TestPip *bool
var PipVirtualEnv *string
var TestPlugins *bool
var timestampAdded bool

func init() {
	RtUrl = flag.String("rt.url", "http://127.0.0.1:8081/artifactory/", "Artifactory url")
	RtUser = flag.String("rt.user", "admin", "Artifactory username")
	RtPassword = flag.String("rt.password", "password", "Artifactory password")
	RtApiKey = flag.String("rt.apikey", "", "Artifactory user API key")
	RtSshKeyPath = flag.String("rt.sshKeyPath", "", "Ssh key file path")
	RtSshPassphrase = flag.String("rt.sshPassphrase", "", "Ssh key passphrase")
	RtAccessToken = flag.String("rt.accessToken", "", "Artifactory access token")
	RtDistributionUrl = flag.String("rt.distUrl", "", "Distribution url")
	RtDistributionAccessToken = flag.String("rt.distAccessToken", "", "Distribution access token")
	TestArtifactory = flag.Bool("test.artifactory", false, "Test Artifactory")
	TestArtifactoryProxy = flag.Bool("test.artifactoryProxy", false, "Test Artifactory proxy")
	TestBintray = flag.Bool("test.bintray", false, "Test Bintray")
	BtUser = flag.String("bt.user", "", "Bintray username")
	BtKey = flag.String("bt.key", "", "Bintray API Key")
	BtOrg = flag.String("bt.org", "", "Bintray organization")
	TestDistribution = flag.Bool("test.distribution", false, "Test distribution")
	TestDocker = flag.Bool("test.docker", false, "Test Docker build")
	TestGo = flag.Bool("test.go", false, "Test Go")
	TestNpm = flag.Bool("test.npm", false, "Test Npm")
	TestGradle = flag.Bool("test.gradle", false, "Test Gradle")
	TestMaven = flag.Bool("test.maven", false, "Test Maven")
	DockerRepoDomain = flag.String("rt.dockerRepoDomain", "", "Docker repository domain")
	DockerVirtualRepo = flag.String("rt.dockerVirtualRepo", "", "Docker virtual repo")
	DockerRemoteRepo = flag.String("rt.dockerRemoteRepo", "", "Docker remote repo")
	DockerLocalRepo = flag.String("rt.DockerLocalRepo", "", "Docker local repo")
	TestNuget = flag.Bool("test.nuget", false, "Test Nuget")
	HideUnitTestLog = flag.Bool("test.hideUnitTestLog", false, "Hide unit tests logs and print it in a file")
	TestPip = flag.Bool("test.pip", false, "Test Pip")
	PipVirtualEnv = flag.String("rt.pipVirtualEnv", "", "Pip virtual-environment path")
	TestPlugins = flag.Bool("test.plugins", false, "Test Plugins")
}

func CleanFileSystem() {
	removeDirs(Out, Temp)
}

func removeDirs(dirs ...string) {
	for _, dir := range dirs {
		isExist, err := fileutils.IsDirExists(dir, false)
		if err != nil {
			log.Error(err)
		}
		if isExist {
			os.RemoveAll(dir)
		}
	}
}

func VerifyExistLocally(expected, actual []string, t *testing.T) {
	if len(actual) == 0 && len(expected) != 0 {
		t.Error("Couldn't find all expected files, expected: " + strconv.Itoa(len(expected)) + ", found: " + strconv.Itoa(len(actual)))
	}
	err := compare(expected, actual)
	assert.NoError(t, err)
}

func ValidateListsIdentical(expected, actual []string) error {
	if len(actual) != len(expected) {
		return errors.New(fmt.Sprintf("Unexpected behavior, \nexpected: [%s], \nfound:    [%s]", strings.Join(expected, ", "), strings.Join(actual, ", ")))
	}
	err := compare(expected, actual)
	return err
}

func ValidateChecksums(filePath string, expectedChecksum fileutils.ChecksumDetails, t *testing.T) {
	localFileDetails, err := fileutils.GetFileDetails(filePath)
	if err != nil {
		t.Error("Couldn't calculate sha1, " + err.Error())
	}
	if localFileDetails.Checksum.Sha1 != expectedChecksum.Sha1 {
		t.Error("sha1 mismatch for "+filePath+", expected: "+expectedChecksum.Sha1, "found: "+localFileDetails.Checksum.Sha1)
	}
	if localFileDetails.Checksum.Md5 != expectedChecksum.Md5 {
		t.Error("md5 mismatch for "+filePath+", expected: "+expectedChecksum.Md5, "found: "+localFileDetails.Checksum.Sha1)
	}
	if localFileDetails.Checksum.Sha256 != expectedChecksum.Sha256 {
		t.Error("sha256 mismatch for "+filePath+", expected: "+expectedChecksum.Sha256, "found: "+localFileDetails.Checksum.Sha1)
	}
}

func compare(expected, actual []string) error {
	for _, v := range expected {
		for i, r := range actual {
			if v == r {
				break
			}
			if i == len(actual)-1 {
				return errors.New("Missing file : " + v)
			}
		}
	}
	return nil
}

func getPathsFromSearchResults(searchResults []artUtils.SearchResult) []string {
	var paths []string
	for _, result := range searchResults {
		paths = append(paths, result.Path)
	}
	return paths
}

func CompareExpectedVsActual(expected []string, actual []artUtils.SearchResult, t *testing.T) {
	actualPaths := getPathsFromSearchResults(actual)
	assert.ElementsMatch(t, expected, actualPaths, fmt.Sprintf("Expected: %v \nActual: %v", expected, actualPaths))
}

func GetTestResourcesPath() string {
	dir, _ := os.Getwd()
	return filepath.ToSlash(dir + "/testdata/")
}

func GetFilePathForBintray(filename, path string, a ...string) string {
	for i := 0; i < len(a); i++ {
		path += a[i] + "/"
	}
	if filename != "" {
		path += filename
	}
	return path
}

func GetFilePathForArtifactory(fileName string) string {
	return GetTestResourcesPath() + "filespecs/" + fileName
}

func GetTestsLogsDir() (string, error) {
	tempDirPath := filepath.Join(coreutils.GetCliPersistentTempDirPath(), "jfrog_tests_logs")
	return tempDirPath, fileutils.CreateDirIfNotExist(tempDirPath)
}

type PackageSearchResultItem struct {
	Name      string
	Path      string
	Package   string
	Version   string
	Repo      string
	Owner     string
	Created   string
	Size      int64
	Sha1      string
	Published bool
}

type JfrogCli struct {
	main        func() error
	prefix      string
	credentials string
}

func NewJfrogCli(mainFunc func() error, prefix, credentials string) *JfrogCli {
	return &JfrogCli{mainFunc, prefix, credentials}
}

func (cli *JfrogCli) Exec(args ...string) error {
	spaceSplit := " "
	os.Args = strings.Split(cli.prefix, spaceSplit)
	output := strings.Split(cli.prefix, spaceSplit)
	for _, v := range args {
		if v == "" {
			continue
		}
		args := strings.Split(v, spaceSplit)
		os.Args = append(os.Args, v)
		output = append(output, args...)
	}
	if cli.credentials != "" {
		args := strings.Split(cli.credentials, spaceSplit)
		os.Args = append(os.Args, args...)
	}

	log.Info("[Command]", strings.Join(output, " "))
	return cli.main()
}

func (cli *JfrogCli) LegacyBuildToolExec(args ...string) error {
	spaceSplit := " "
	os.Args = strings.Split(cli.prefix, spaceSplit)
	os.Args = append(os.Args, args...)

	log.Info("[Command]", os.Args)

	if cli.credentials != "" {
		args := strings.Split(cli.credentials, spaceSplit)
		os.Args = append(os.Args, args...)
	}
	return cli.main()
}

func (cli *JfrogCli) WithoutCredentials() *JfrogCli {
	return &JfrogCli{cli.main, cli.prefix, ""}
}

type gitManager struct {
	dotGitPath string
}

func GitExecutor(dotGitPath string) *gitManager {
	return &gitManager{dotGitPath: dotGitPath}
}

func (m *gitManager) GetUrl() (string, string, error) {
	return m.execGit("config", "--get", "remote.origin.url")
}

func (m *gitManager) GetRevision() (string, string, error) {
	return m.execGit("show", "-s", "--format=%H", "HEAD")
}

func (m *gitManager) GetBranch() (string, string, error) {
	return m.execGit("branch", "--show-current")
}

func (m *gitManager) GetMessage(revision string) (string, string, error) {
	return m.execGit("show", "-s", "--format=%B", revision)
}

func (m *gitManager) execGit(args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = m.dotGitPath
	cmd.Stdin = nil
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	errorutils.CheckError(err)
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func DeleteFiles(deleteSpec *spec.SpecFiles, serverDetails *config.ServerDetails) (successCount, failCount int, err error) {
	deleteCommand := generic.NewDeleteCommand()
	deleteCommand.SetThreads(3).SetSpec(deleteSpec).SetServerDetails(serverDetails).SetDryRun(false)
	reader, err := deleteCommand.GetPathsToDelete()
	if err != nil {
		return 0, 0, err
	}
	defer reader.Close()
	return deleteCommand.DeleteFiles(reader)
}

// This function makes no assertion, caller is responsible to assert as needed.
func GetBuildInfo(serverDetails *config.ServerDetails, buildName, buildNumber string) (pbi *buildinfo.PublishedBuildInfo, found bool, err error) {
	servicesManager, err := artUtils.CreateServiceManager(serverDetails, -1, false)
	if err != nil {
		return nil, false, err
	}
	params := services.NewBuildInfoParams()
	params.BuildName = buildName
	params.BuildNumber = buildNumber
	return servicesManager.GetBuildInfo(params)
}

var reposConfigMap = map[*string]string{
	&DistRepo1:        DistributionRepoConfig1,
	&DistRepo2:        DistributionRepoConfig2,
	&DockerRepo:       DockerRepoConfig,
	&GoRepo:           GoLocalRepositoryConfig,
	&GradleRepo:       GradleRepositoryConfig,
	&MvnRepo1:         MavenRepositoryConfig1,
	&MvnRepo2:         MavenRepositoryConfig2,
	&MvnRemoteRepo:    MavenRemoteRepositoryConfig,
	&GradleRemoteRepo: GradleRemoteRepositoryConfig,
	&NpmRepo:          NpmLocalRepositoryConfig,
	&NpmRemoteRepo:    NpmRemoteRepositoryConfig,
	&PypiRemoteRepo:   PypiRemoteRepositoryConfig,
	&PypiVirtualRepo:  PypiVirtualRepositoryConfig,
	&RtDebianRepo:     DebianTestRepositoryConfig,
	&RtLfsRepo:        GitLfsTestRepositoryConfig,
	&RtRepo1:          Repo1RepositoryConfig,
	&RtRepo2:          Repo2RepositoryConfig,
	&RtVirtualRepo:    VirtualRepositoryConfig,
}

var CreatedNonVirtualRepositories map[*string]string
var CreatedVirtualRepositories map[*string]string

func getNeededRepositories(reposMap map[*bool][]*string) map[*string]string {
	reposToCreate := map[*string]string{}
	for needed, testRepos := range reposMap {
		if *needed {
			for _, repo := range testRepos {
				reposToCreate[repo] = reposConfigMap[repo]
			}
		}
	}
	return reposToCreate
}

func getNeededBuildNames(buildNamesMap map[*bool][]*string) []string {
	var neededBuildNames []string
	for needed, buildNames := range buildNamesMap {
		if *needed {
			for _, buildName := range buildNames {
				neededBuildNames = append(neededBuildNames, *buildName)
			}
		}
	}
	return neededBuildNames
}

// Return local and remote repositories for the test suites, respectfully
func GetNonVirtualRepositories() map[*string]string {
	nonVirtualReposMap := map[*bool][]*string{
		TestArtifactory:  {&RtRepo1, &RtRepo2, &RtLfsRepo, &RtDebianRepo},
		TestDistribution: {&DistRepo1, &DistRepo2},
		TestDocker:       {&DockerRepo},
		TestGo:           {&GoRepo},
		TestGradle:       {&GradleRepo, &GradleRemoteRepo},
		TestMaven:        {&MvnRepo1, &MvnRepo2, &MvnRemoteRepo},
		TestNpm:          {&NpmRepo, &NpmRemoteRepo},
		TestNuget:        {},
		TestPip:          {&PypiRemoteRepo},
		TestPlugins:      {&RtRepo1},
	}
	return getNeededRepositories(nonVirtualReposMap)
}

// Return virtual repositories for the test suites, respectfully
func GetVirtualRepositories() map[*string]string {
	virtualReposMap := map[*bool][]*string{
		TestArtifactory:  {&RtVirtualRepo},
		TestDistribution: {},
		TestDocker:       {},
		TestGo:           {},
		TestGradle:       {},
		TestMaven:        {},
		TestNpm:          {},
		TestNuget:        {},
		TestPip:          {&PypiVirtualRepo},
	}
	return getNeededRepositories(virtualReposMap)
}

func GetAllRepositoriesNames() []string {
	var baseRepoNames []string
	for repoName := range GetNonVirtualRepositories() {
		baseRepoNames = append(baseRepoNames, *repoName)
	}
	for repoName := range GetVirtualRepositories() {
		baseRepoNames = append(baseRepoNames, *repoName)
	}
	return baseRepoNames
}

func GetTestUsersNames() []string {
	return []string{UserName1, UserName2}
}

func GetBuildNames() []string {
	buildNamesMap := map[*bool][]*string{
		TestArtifactory:  {&RtBuildName1, &RtBuildName2, &RtBuildNameWithSpecialChars},
		TestDistribution: {},
		TestDocker:       {&DockerBuildName},
		TestGo:           {&GoBuildName},
		TestGradle:       {&GradleBuildName},
		TestMaven:        {},
		TestNpm:          {&NpmBuildName, &YarnBuildName},
		TestNuget:        {&NuGetBuildName},
		TestPip:          {&PipBuildName},
	}
	return getNeededBuildNames(buildNamesMap)
}

// Builds and repositories names to replace in the test files.
// We use substitution map to set repositories and builds with timestamp.
func getSubstitutionMap() map[string]string {
	return map[string]string{
		"${REPO1}":                     RtRepo1,
		"${REPO2}":                     RtRepo2,
		"${REPO_1_AND_2}":              RtRepo1And2,
		"${VIRTUAL_REPO}":              RtVirtualRepo,
		"${LFS_REPO}":                  RtLfsRepo,
		"${DEBIAN_REPO}":               RtDebianRepo,
		"${DOCKER_REPO}":               DockerRepo,
		"${DOCKER_REPO_DOMAIN}":        *DockerRepoDomain,
		"${MAVEN_REPO1}":               MvnRepo1,
		"${MAVEN_REPO2}":               MvnRepo2,
		"${MAVEN_REMOTE_REPO}":         MvnRemoteRepo,
		"${GRADLE_REMOTE_REPO}":        GradleRemoteRepo,
		"${GRADLE_REPO}":               GradleRepo,
		"${NPM_REPO}":                  NpmRepo,
		"${NPM_REMOTE_REPO}":           NpmRemoteRepo,
		"${GO_REPO}":                   GoRepo,
		"${RT_SERVER_ID}":              RtServerId,
		"${RT_URL}":                    *RtUrl,
		"${RT_API_KEY}":                *RtApiKey,
		"${RT_USERNAME}":               *RtUser,
		"${RT_PASSWORD}":               *RtPassword,
		"${RT_CREDENTIALS_BASIC_AUTH}": base64.StdEncoding.EncodeToString([]byte(*RtUser + ":" + *RtPassword)),
		"${RT_ACCESS_TOKEN}":           *RtAccessToken,
		"${PYPI_REMOTE_REPO}":          PypiRemoteRepo,
		"${PYPI_VIRTUAL_REPO}":         PypiVirtualRepo,
		"${BUILD_NAME1}":               RtBuildName1,
		"${BUILD_NAME2}":               RtBuildName2,
		"${BINTRAY_REPO}":              BintrayRepo,
		"${BUNDLE_NAME}":               BundleName,
		"${DIST_REPO1}":                DistRepo1,
		"${DIST_REPO2}":                DistRepo2,
		"{USER_NAME_1}":                UserName1,
		"{PASSWORD_1}":                 Password1,
		"{USER_NAME_2}":                UserName2,
		"{PASSWORD_2}":                 Password2,
	}
}

// Add timestamp to builds and repositories names
func AddTimestampToGlobalVars() {
	// Make sure the global timestamp is added only once even in case of multiple tests flags
	if timestampAdded {
		return
	}
	timestampSuffix := "-" + strconv.FormatInt(time.Now().Unix(), 10)
	// Repositories
	BintrayRepo += timestampSuffix
	DockerRepo += timestampSuffix
	DistRepo1 += timestampSuffix
	DistRepo2 += timestampSuffix
	GoRepo += timestampSuffix
	GradleRemoteRepo += timestampSuffix
	GradleRepo += timestampSuffix
	MvnRemoteRepo += timestampSuffix
	MvnRepo1 += timestampSuffix
	MvnRepo2 += timestampSuffix
	NpmRepo += timestampSuffix
	NpmRemoteRepo += timestampSuffix
	PypiRemoteRepo += timestampSuffix
	PypiVirtualRepo += timestampSuffix
	RtDebianRepo += timestampSuffix
	RtLfsRepo += timestampSuffix
	RtRepo1 += timestampSuffix
	RtRepo1And2 += timestampSuffix
	RtRepo1And2Placeholder += timestampSuffix
	RtRepo2 += timestampSuffix
	RtVirtualRepo += timestampSuffix

	// Builds/bundles/images
	BundleName += timestampSuffix
	DockerBuildName += timestampSuffix
	DockerImageName += timestampSuffix
	DotnetBuildName += timestampSuffix
	GoBuildName += timestampSuffix
	GradleBuildName += timestampSuffix
	NpmBuildName += timestampSuffix
	YarnBuildName += timestampSuffix
	NuGetBuildName += timestampSuffix
	PipBuildName += timestampSuffix
	RtBuildName1 += timestampSuffix
	RtBuildName2 += timestampSuffix
	RtBuildNameWithSpecialChars += timestampSuffix
	RtPermissionTargetName += timestampSuffix

	// Users
	UserName1 += timestampSuffix
	UserName2 += timestampSuffix
	rand.Seed(time.Now().Unix())
	Password1 += timestampSuffix + strconv.FormatFloat(rand.Float64(), 'f', 2, 32)
	Password2 += timestampSuffix + strconv.FormatFloat(rand.Float64(), 'f', 2, 32)

	timestampAdded = true
}

func ReplaceTemplateVariables(path, destPath string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errorutils.CheckError(err)
	}

	for name, value := range getSubstitutionMap() {
		content = bytes.Replace(content, []byte(name), []byte(value), -1)
	}
	if destPath == "" {
		destPath, err = os.Getwd()
		if err != nil {
			return "", errorutils.CheckError(err)
		}
		destPath = filepath.Join(destPath, Temp)
	}
	err = os.MkdirAll(destPath, 0700)
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	specPath := filepath.Join(destPath, filepath.Base(path))
	log.Info("Creating spec file at:", specPath)
	err = ioutil.WriteFile(specPath, []byte(content), 0700)
	if err != nil {
		return "", errorutils.CheckError(err)
	}

	return specPath, nil
}

func CreateSpec(fileName string) (string, error) {
	searchFilePath := GetFilePathForArtifactory(fileName)
	searchFilePath, err := ReplaceTemplateVariables(searchFilePath, "")
	return searchFilePath, err
}

func ConvertSliceToMap(props []utils.Property) map[string][]string {
	propsMap := make(map[string][]string)
	for _, item := range props {
		propsMap[item.Key] = append(propsMap[item.Key], item.Value)
	}
	return propsMap
}

// Set user and password from access token.
// Return the original user and password to allow restoring them in the end of the test.
func SetBasicAuthFromAccessToken(t *testing.T) (string, string) {
	var err error
	origUser := *RtUser
	origPassword := *RtPassword

	*RtUser, err = auth.ExtractUsernameFromAccessToken(*RtAccessToken)
	assert.NoError(t, err)
	*RtPassword = *RtAccessToken
	return origUser, origPassword
}

// Clean items with timestamp older than 24 hours. Used to delete old repositories, builds, release bundles and Docker images.
// baseItemNames - The items to delete without timestamp, i.e. [cli-tests-rt1, cli-tests-rt2, ...]
// getActualItems - Function that returns all actual items in the remote server, i.e. [cli-tests-rt1-1592990748, cli-tests-rt2-1592990748, ...]
// deleteItem - Function that deletes the item by name
func CleanUpOldItems(baseItemNames []string, getActualItems func() ([]string, error), deleteItem func(string)) {
	actualItems, err := getActualItems()
	if err != nil {
		log.Warn("Couldn't retrieve items", err)
		return
	}
	now := time.Now()
	for _, baseItemName := range baseItemNames {
		itemPattern := regexp.MustCompile(`^` + baseItemName + `-(\d*)$`)
		for _, item := range actualItems {
			regexGroups := itemPattern.FindStringSubmatch(item)
			if regexGroups == nil {
				// Item does not match
				continue
			}

			itemTimestamp, err := strconv.ParseInt(regexGroups[len(regexGroups)-1], 10, 64)
			if err != nil {
				log.Warn("Error while parsing timestamp of ", item, err)
				continue
			}

			itemTime := time.Unix(itemTimestamp, 0)
			if now.Sub(itemTime).Hours() > 24 {
				deleteItem(item)
			}
		}
	}
}

// Set new logger with output redirection to a buffer.
// Caller is responsible to set the old log back.
func RedirectLogOutputToBuffer() (buffer *bytes.Buffer, previousLog log.Log) {
	previousLog = log.Logger
	newLog := log.NewLogger(corelog.GetCliLogLevel(), nil)
	buffer = &bytes.Buffer{}
	newLog.SetOutputWriter(buffer)
	log.SetLogger(newLog)
	return buffer, previousLog
}

// Set new logger with output redirection to a null logger. This is useful for negative tests.
// Caller is responsible to set the old log back.
func RedirectLogOutputToNil() (previousLog log.Log) {
	previousLog = log.Logger
	newLog := log.NewLogger(corelog.GetCliLogLevel(), nil)
	newLog.SetOutputWriter(ioutil.Discard)
	newLog.SetLogsWriter(ioutil.Discard)
	log.SetLogger(newLog)
	return previousLog
}

func VerifySha256DetailedSummaryFromBuffer(t *testing.T, buffer *bytes.Buffer, logger log.Log) {
	content := buffer.Bytes()
	buffer.Reset()
	logger.Output(string(content))

	var result summary.BuildInfoSummary
	err := json.Unmarshal(content, &result)
	assert.NoError(t, err)

	assert.Equal(t, summary.Success, result.Status)
	assert.True(t, result.Totals.Success > 0)
	assert.Equal(t, 0, result.Totals.Failure)
	// Verify a sha256 was returned
	assert.NotEmpty(t, result.Sha256Array, "Summary validation failed - no sha256 has returned from Artifactory.")
	for _, sha256 := range result.Sha256Array {
		// Verify sha256 is valid (a string size 256 characters) and not an empty string.
		assert.Equal(t, 64, len(sha256.Sha256Str), "Summary validation failed - invalid sha256 has returned from artifactory")
	}
}

func VerifySha256DetailedSummaryFromResult(t *testing.T, result *commandutils.Result) {
	result.Reader()
	reader := result.Reader()
	defer reader.Close()
	assert.NoError(t, reader.GetError())
	for transferDetails := new(clientutils.FileTransferDetails); reader.NextRecord(transferDetails) == nil; transferDetails = new(clientutils.FileTransferDetails) {
		assert.Equal(t, 64, len(transferDetails.Sha256), "Summary validation failed - invalid sha256 has returned from artifactory")
	}
}
