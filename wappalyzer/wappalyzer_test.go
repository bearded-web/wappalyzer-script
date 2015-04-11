package wappalyzer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"golang.org/x/net/context"
	"github.com/bearded-web/bearded/models/plan"
	"github.com/bearded-web/bearded/models/report"
	"github.com/bearded-web/bearded/pkg/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ClientMock struct {
	mock.Mock
	*script.FakeClient
}

func (m *ClientMock) GetPlugin(ctx context.Context, name string) (*script.Plugin, error) {
	return script.NewPlugin(name, m, "0.0.2"), nil
}

func (m *ClientMock) RunPlugin(ctx context.Context, conf *plan.WorkflowStep) (*report.Report, error) {
	args := m.Called(ctx, conf)
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *ClientMock) SendReport(ctx context.Context, rep *report.Report) error {
	data, _ := json.Marshal(rep)
	println(string(data))
	args := m.Called(ctx, rep)
	return args.Error(0)
}

func TestParseWappalyzer(t *testing.T) {
	rep := loadReport("wapp-report1.json")
	require.Equal(t, rep.Type, report.TypeRaw)
	w := New()
	techs, err := w.parseWappalyzer(rep.Raw.Raw)
	require.NoError(t, err)
	require.Equal(t, 5, len(techs))
	assert.Equal(t, techs[0].Name, "AngularJS")
	assert.Equal(t, techs[1].Name, "Google Analytics")
	assert.Equal(t, techs[2].Name, "Lo-dash")
	assert.Equal(t, techs[3].Name, "Nginx")
	assert.Equal(t, techs[4].Name, "Twitter Bootstrap")
}

func TestHandle(t *testing.T) {
	target := "http://example.com"
	bg := context.Background()
	testData := []struct {
		ToolReport     *report.Report
		ExpectedReport *report.Report
	}{
		{loadReport("wapp-report1.json"), loadReport("report1.json")},
	}

	for _, data := range testData {
		client := &ClientMock{}
		pl := plan.WorkflowStep{
			Name:   "underscan",
			Plugin: "barbudo/wappalyzer:0.0.2",
			Conf:   &plan.Conf{CommandArgs: fmt.Sprintf(target)},
		}
		client.On("RunPlugin", bg, &pl).
			Return(data.ToolReport, nil).Once()
		client.On("SendReport", bg, data.ExpectedReport).Return(nil).Once()

		var s script.Scripter = New()
		err := s.Handle(bg, client, &plan.Conf{Target: target, CommandArgs: fmt.Sprintf(target)})
		require.NoError(t, err)
		client.Mock.AssertExpectations(t)
	}
}

// test data
const testDataDir = "test_data"

func loadTestData(filename string) string {
	file := path.Join(testDataDir, filename)
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func loadReport(filename string) *report.Report {
	rep := report.Report{}
	if err := json.Unmarshal([]byte(loadTestData(filename)), &rep); err != nil {
		panic(err)
	}
	return &rep
}
