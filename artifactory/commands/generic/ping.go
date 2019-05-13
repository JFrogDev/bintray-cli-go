package generic

import (
	"github.com/jfrog/jfrog-cli-go/artifactory/utils"
	"github.com/jfrog/jfrog-cli-go/utils/config"
)

type PingCommand struct {
	rtDetails *config.ArtifactoryDetails
	response  []byte
}

func (pc *PingCommand) Response() []byte {
	return pc.response
}

func (pc *PingCommand) RtDetails() *config.ArtifactoryDetails {
	return pc.rtDetails
}

func (pc *PingCommand) SetRtDetails(rtDetails *config.ArtifactoryDetails) *PingCommand {
	pc.rtDetails = rtDetails
	return pc
}

func (pc *PingCommand) CommandName() string {
	return "rt_ping"
}

func (pc *PingCommand) Run() error {
	var err error
	pc.response, err = pc.Ping()
	if err != nil {
		return err
	}
	return nil
}

func (pc *PingCommand) Ping() ([]byte, error) {
	servicesManager, err := utils.CreateServiceManager(pc.rtDetails, false)
	if err != nil {
		return nil, err
	}
	return servicesManager.Ping()
}
