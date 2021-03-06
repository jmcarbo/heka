package plugins

import (
	"errors"
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	ts "github.com/mozilla-services/heka/pipeline/testsupport"
	plugins_ts "github.com/mozilla-services/heka/plugins/testsupport"
	"github.com/mozilla-services/heka/sandbox"
	"github.com/rafrombrc/gomock/gomock"
	gs "github.com/rafrombrc/gospec/src/gospec"
	"io/ioutil"
	"os"
	"time"
)

func OutputSpec(c gs.Context) {
	t := new(ts.SimpleT)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oth := plugins_ts.NewOutputTestHelper(ctrl)
	pConfig := pipeline.NewPipelineConfig(nil)
	inChan := make(chan *pipeline.PipelinePack, 1)
	timer := make(chan time.Time, 1)

	c.Specify("A SandboxOutput", func() {
		output := new(SandboxOutput)
		output.SetPipelineConfig(pConfig)
		conf := output.ConfigStruct().(*sandbox.SandboxConfig)
		conf.ScriptFilename = "../lua/testsupport/output.lua"
		conf.ModuleDirectory = "../lua/modules"
		supply := make(chan *pipeline.PipelinePack, 1)
		pack := pipeline.NewPipelinePack(supply)
		data := "1376389920 debug id=2321 url=example.com item=1"
		pack.Message.SetPayload(data)
		err := output.Init(conf)
		c.Expect(err, gs.IsNil)

		oth.MockOutputRunner.EXPECT().InChan().Return(inChan)
		oth.MockOutputRunner.EXPECT().Ticker().Return(timer)

		c.Specify("writes a payload to file", func() {
			inChan <- pack
			close(inChan)
			err = output.Run(oth.MockOutputRunner, oth.MockHelper)
			c.Assume(err, gs.IsNil)

			tmpFile, err := os.Open("output.lua.txt")
			defer tmpFile.Close()
			c.Assume(err, gs.IsNil)
			contents, err := ioutil.ReadAll(tmpFile)
			c.Assume(err, gs.IsNil)
			c.Expect(string(contents), gs.Equals, data)
			c.Expect(output.processMessageCount, gs.Equals, int64(1))
			c.Expect(output.processMessageSamples, gs.Equals, int64(1))
			c.Expect(output.processMessageFailures, gs.Equals, int64(0))
			c.Expect(output.processMessageDuration, gs.Not(gs.Equals), int64(0))
		})

		c.Specify("failure processing data", func() {
			oth.MockOutputRunner.EXPECT().LogError(fmt.Errorf("failure message"))
			pack.Message.SetPayload("FAILURE")
			inChan <- pack
			close(inChan)
			err = output.Run(oth.MockOutputRunner, oth.MockHelper)
			c.Assume(err, gs.IsNil)
			c.Expect(output.processMessageFailures, gs.Equals, int64(1))
		})

		c.Specify("user abort processing data", func() {
			e := errors.New("FATAL: user abort")
			oth.MockOutputRunner.EXPECT().LogError(e)
			pack.Message.SetPayload("USERABORT")
			inChan <- pack
			err = output.Run(oth.MockOutputRunner, oth.MockHelper)
			c.Expect(err.Error(), gs.Equals, e.Error())
		})

		c.Specify("fatal error processing data", func() {
			oth.MockOutputRunner.EXPECT().LogError(gomock.Any())
			pack.Message.SetPayload("FATAL")
			inChan <- pack
			err = output.Run(oth.MockOutputRunner, oth.MockHelper)
			c.Expect(err, gs.Not(gs.IsNil))
		})

		c.Specify("fatal error in timer_event", func() {
			oth.MockOutputRunner.EXPECT().LogError(gomock.Any())
			timer <- time.Now()
			err = output.Run(oth.MockOutputRunner, oth.MockHelper)
			c.Expect(err, gs.Not(gs.IsNil))
		})
	})
}
