package wappalyzer

import (
	"encoding/json"

	"golang.org/x/net/context"
	"github.com/facebookgo/stackerr"

	"github.com/bearded-web/bearded/models/plan"
	"github.com/bearded-web/bearded/models/report"
	"github.com/bearded-web/bearded/models/tech"
	"github.com/bearded-web/bearded/pkg/script"
	"github.com/davecgh/go-spew/spew"
)

const toolName = "barbudo/wappalyzer"

//var supportedVersions = []string{
//	"0.0.2",
//}

type WappalyzerResult struct {
	Applications []*WappalyzerItem `json:"applications"`
}

type WappalyzerItem struct {
	Name       string          `json:"name"`
	Confidence int             `json:"confidence"`
	Version    string          `json:"version"`
	Categories []tech.Category `json:"categories"`
}

type Wappalyzer struct {
}

func New() *Wappalyzer {
	return &Wappalyzer{}
}

func (s *Wappalyzer) Handle(ctx context.Context, client script.ClientV1, conf *plan.Conf) error {
	// Check if retirejs plugin is available
	println("get tool")
	pl, err := s.getTool(ctx, client)
	if err != nil {
		return err
	}

	println("run wappalyzer")
	// Run wappalyzer util
	rep, err := pl.Run(ctx, pl.LatestVersion(), &plan.Conf{CommandArgs: conf.Target})
	if err != nil {
		return stackerr.Wrap(err)
	}
	println("wappalyzer finished")
	// Get and parse wappalyzer output
	if rep.Type != report.TypeRaw {
		return stackerr.Newf("Wappalyzer report type should be TypeRaw, but got %s instead", rep.Type)
	}
	resultReport := report.Report{Type: report.TypeEmpty}

	techs, err := s.parseWappalyzer(rep.Raw.Raw)
	if err != nil {
		return stackerr.Wrap(err)
	}
	if len(techs) > 0 {
		resultReport = report.Report{
			Type:  report.TypeTechs,
			Techs: techs,
		}
	}
	println("send report")
	// push reports
	client.SendReport(ctx, &resultReport)
	spew.Dump(resultReport)
	println("sent")
	// exit
	return nil
}

func (s *Wappalyzer) parseWappalyzer(data string) ([]*tech.Tech, error) {
	application := WappalyzerResult{}
	err := json.Unmarshal([]byte(data), &application)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	techs := []*tech.Tech{}
	for _, item := range application.Applications {
		tech := tech.Tech{
			Name:       item.Name,
			Version:    item.Version,
			Confidence: item.Confidence,
			Categories: item.Categories,
		}
		techs = append(techs, &tech)
	}
	return techs, nil
}

// Check if wappalyzer plugin is available
func (s *Wappalyzer) getTool(ctx context.Context, client script.ClientV1) (*script.Plugin, error) {
	pl, err := client.GetPlugin(ctx, toolName)
	if err != nil {
		return nil, err
	}
	return pl, err
	//	pl.LatestSupportedVersion(supportedVersions)
}
