package utils

import (
	"github.com/jfrogdev/jfrog-cli-go/utils/io/httputils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"github.com/jfrogdev/jfrog-cli-go/errors/httperrors"
)

func SetProps(spec *SpecFiles, flags CommonFlags, props string) error {
	err := PreCommandSetup(flags)
	if err != nil {
		return err
	}
	resultItems, err := SearchBySpecFiles(spec, flags)
	if err != nil {
		return err
	}
	updatePropertiesBaseUrl := flags.GetArtifactoryDetails().Url + "api/storage"
	log.Info("Setting properties...")
	encodedParam, err := EncodeParams(props)
	if err != nil {
		return err
	}
	for _, item := range resultItems {
		log.Info("Setting properties on", item.GetFullUrl())
		httpClientsDetails := GetArtifactoryHttpClientDetails(flags.GetArtifactoryDetails())
		setPropertiesUrl := updatePropertiesBaseUrl + "/" + item.GetFullUrl() + "?properties=" + encodedParam
		log.Debug("Sending set properties request:", setPropertiesUrl)
		resp, body, err := httputils.SendPut(setPropertiesUrl, nil, httpClientsDetails)
		if err != nil {
			return err
		}
		if err = httperrors.CheckResponseStatus(resp, body, 204); err != nil {
			return err
		}
	}

	log.Info("Done setting properties.")
	return err
}