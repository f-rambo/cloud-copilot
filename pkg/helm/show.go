package helm

import (
	"fmt"

	"github.com/f-rambo/ocean/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	pkgChart "helm.sh/helm/v3/pkg/chart"
)

type ChartInfo struct {
	Name        string
	Config      string
	Readme      string
	Description string
	Metadata    pkgChart.Metadata
	Version     string
	AppName     string
}

func GetLocalChartInfo(appName, path, chart string) (*ChartInfo, error) {
	if chart == "" {
		return nil, errors.New("chart is empty")
	}
	if utils.IsHttpUrl(chart) {
		path = fmt.Sprintf("%s%s/", path, appName)
		fileName := utils.GetFileNameByUrl(chart)
		err := utils.DownloadFile(chart, path, fileName)
		if err != nil {
			return nil, errors.WithMessage(err, "download chart fail")
		}
		chart = fileName
	}
	chartPath := path + chart
	readme, err := action.NewShow(action.ShowReadme).Run(chartPath)
	if err != nil {
		return nil, errors.WithMessage(err, "show readme fail")
	}
	chartYaml, err := action.NewShow(action.ShowChart).Run(chartPath)
	if err != nil {
		return nil, errors.WithMessage(err, "show chart fail")
	}
	valuesYaml, err := action.NewShow(action.ShowValues).Run(chartPath)
	if err != nil {
		return nil, errors.WithMessage(err, "show values fail")
	}
	chartMateData := &pkgChart.Metadata{}
	err = yaml.Unmarshal([]byte(chartYaml), chartMateData)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal chart yaml fail")
	}
	chartInfo := &ChartInfo{
		Name:        chartMateData.Name,
		Config:      valuesYaml,
		Readme:      readme,
		Description: chartMateData.Description,
		Metadata:    *chartMateData,
		Version:     chartMateData.Version,
		AppName:     chartMateData.Name,
	}
	return chartInfo, nil
}
