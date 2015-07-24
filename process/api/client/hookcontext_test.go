package client_test

import (
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v5"

	"github.com/juju/juju/process/api"
	"github.com/juju/juju/process/api/client"
)

type clientSuite struct {
	stub       *testing.Stub
	facade     *stubFacade
	tag        string
	process    api.Process
	definition api.ProcessDefinition
}

var _ = gc.Suite(&clientSuite{})

func (s *clientSuite) SetUpTest(c *gc.C) {
	s.stub = &testing.Stub{}
	s.facade = &stubFacade{stub: s.stub}
	s.tag = "machine-tag"
	s.definition = api.ProcessDefinition{
		Name:        "foobar",
		Description: "desc",
		Type:        "type",
		TypeOptions: map[string]string{"foo": "bar"},
		Command:     "cmd",
		Image:       "img",
		Ports: []api.ProcessPort{{
			External: 8080,
			Internal: 80,
			Endpoint: "endpoint",
		}},
		Volumes: []api.ProcessVolume{{
			ExternalMount: "/foo/bar",
			InternalMount: "/baz/bat",
			Mode:          "ro",
			Name:          "volname",
		}},
		EnvVars: map[string]string{"envfoo": "bar"},
	}

	s.process = api.Process{
		Definition: s.definition,
		Details: api.ProcessDetails{
			ID:     "idfoo",
			Status: api.ProcessStatus{Label: "process status"},
		},
	}

}

func (s *clientSuite) TestAllDefinitions(c *gc.C) {
	s.facade.FacadeCallFn = func(name string, params, response interface{}) error {
		results := response.(*api.ListDefinitionsResults)
		*results = api.ListDefinitionsResults{
			Results: []api.ProcessDefinition{s.definition},
		}
		return nil
	}
	pclient := client.NewHookContextClient(s.facade)

	definitions, err := pclient.AllDefinitions()
	c.Assert(err, jc.ErrorIsNil)

	c.Check(definitions, jc.DeepEquals, []charm.Process{{
		Name:        "foobar",
		Description: "desc",
		Type:        "type",
		TypeOptions: map[string]string{"foo": "bar"},
		Command:     "cmd",
		Image:       "img",
		Ports: []charm.ProcessPort{{
			External: 8080,
			Internal: 80,
			Endpoint: "endpoint",
		}},
		Volumes: []charm.ProcessVolume{{
			ExternalMount: "/foo/bar",
			InternalMount: "/baz/bat",
			Mode:          "ro",
			Name:          "volname",
		}},
		EnvVars: map[string]string{"envfoo": "bar"},
	}})
	s.stub.CheckCallNames(c, "FacadeCall")
}

func (s *clientSuite) TestRegisterProcesses(c *gc.C) {
	numStubCalls := 0
	s.facade.FacadeCallFn = func(name string, params, response interface{}) error {
		numStubCalls++
		c.Check(name, gc.Equals, "RegisterProcesses")

		typedResponse, ok := response.(*api.ProcessResults)
		c.Assert(ok, gc.Equals, true)

		typedResponse.Results = append(typedResponse.Results, api.ProcessResult{
			ID:    "idfoo",
			Error: nil,
		})

		return nil
	}

	pclient := client.NewHookContextClient(s.facade)

	ids, err := pclient.RegisterProcesses(s.process)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(len(ids), gc.Equals, 1)
	c.Check(numStubCalls, gc.Equals, 1)
	c.Check(ids[0], gc.Equals, api.ProcessResult{ID: s.process.Details.ID, Error: nil})
}

func (s *clientSuite) TestListAllProcesses(c *gc.C) {
	numStubCalls := 0

	s.facade.FacadeCallFn = func(name string, params, response interface{}) error {
		numStubCalls++
		c.Check(name, gc.Equals, "ListProcesses")

		typedResponse, ok := response.(*api.ListProcessesResults)
		c.Assert(ok, gc.Equals, true)

		result := api.ListProcessResult{ID: s.process.Details.ID, Info: s.process, Error: nil}
		typedResponse.Results = append(typedResponse.Results, result)

		return nil
	}
	pclient := client.NewHookContextClient(s.facade)

	processes, err := pclient.ListProcesses(s.tag)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(len(processes), gc.Equals, 1)
	c.Check(numStubCalls, gc.Equals, 1)
	c.Check(processes[0], gc.DeepEquals, api.ListProcessResult{ID: "idfoo", Info: s.process, Error: nil})
}

func (s *clientSuite) TestSetProcessesStatus(c *gc.C) {
	numStubCalls := 0

	s.facade.FacadeCallFn = func(name string, params, response interface{}) error {
		numStubCalls++
		c.Check(name, gc.Equals, "SetProcessesStatus")

		typedParams, ok := params.(*api.SetProcessesStatusArgs)
		c.Assert(ok, gc.Equals, true)

		c.Check(len(typedParams.Args), gc.Equals, 1)

		arg := typedParams.Args[0]
		c.Check(arg.ID, gc.Equals, "idfoo")
		c.Check(arg.Status, gc.DeepEquals, api.ProcessStatus{Label: "Running"})

		return nil
	}

	pclient := client.NewHookContextClient(s.facade)
	err := pclient.SetProcessesStatus("Running", "idfoo")
	c.Assert(err, jc.ErrorIsNil)

	c.Check(numStubCalls, gc.Equals, 1)
}

func (s *clientSuite) TestUnregisterProcesses(c *gc.C) {
	numStubCalls := 0

	s.facade.FacadeCallFn = func(name string, params, response interface{}) error {
		numStubCalls++
		c.Check(name, gc.Equals, "UnregisterProcesses")

		typedParams, ok := params.(*api.UnregisterProcessesArgs)
		c.Assert(ok, gc.Equals, true)

		c.Check(len(typedParams.IDs), gc.Equals, 1)

		return nil
	}

	pclient := client.NewHookContextClient(s.facade)
	err := pclient.UnregisterProcesses(s.tag)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(numStubCalls, gc.Equals, 1)
}

type stubFacade struct {
	stub         *testing.Stub
	FacadeCallFn func(name string, params, response interface{}) error
}

func (s *stubFacade) FacadeCall(request string, params, response interface{}) error {
	s.stub.AddCall("FacadeCall", request, params, response)
	if err := s.stub.NextErr(); err != nil {
		return err
	}

	if s.FacadeCallFn != nil {
		return s.FacadeCallFn(request, params, response)
	}
	return nil
}