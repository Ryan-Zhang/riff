/*
 * Copyright 2018 The original author or authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package commands_test

import (
	"fmt"

	"strings"

	v1alpha12 "github.com/knative/eventing/pkg/apis/channels/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectriff/riff/cmd/commands"
	"github.com/projectriff/riff/pkg/core"
	"github.com/projectriff/riff/pkg/core/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("The riff function command", func() {
	Context("when given wrong args or flags", func() {
		var (
			mockClient core.Client
			fc         *cobra.Command
		)
		BeforeEach(func() {
			mockClient = nil
			fc = commands.FunctionCreate(&mockClient)
		})
		It("should fail with no args", func() {
			fc.SetArgs([]string{})
			err := fc.Execute()
			Expect(err).To(MatchError("accepts 2 arg(s), received 0"))
		})
		It("should fail with invalid invoker or function name", func() {
			fc.SetArgs([]string{".invalid", "fn-name"})
			err := fc.Execute()
			Expect(err).To(MatchError(ContainSubstring("must start and end with an alphanumeric character")))

			fc = commands.FunctionCreate(&mockClient)
			fc.SetArgs([]string{"node", "invålid"})
			err = fc.Execute()
			Expect(err).To(MatchError(ContainSubstring("must start and end with an alphanumeric character")))
		})
		It("should fail without required flags", func() {
			fc.SetArgs([]string{"node", "square"})
			err := fc.Execute()
			Expect(err).To(MatchError(ContainSubstring("required flag(s)")))
			Expect(err).To(MatchError(ContainSubstring("git-repo")))
			Expect(err).To(MatchError(ContainSubstring("image")))
		})
		It("should fail when input is set w/o bus or cluster-bus", func() {
			fc.SetArgs([]string{"node", "square", "--image", "foo/bar", "--git-repo", "https://github.com/repo",
				"--input", "i"})
			err := fc.Execute()
			Expect(err).To(MatchError("when --input is set, at least one of --bus, --cluster-bus must be set"))
		})
	})

	Context("when given suitable args and flags", func() {
		var (
			client core.Client
			asMock *mocks.Client
			fc     *cobra.Command
		)
		BeforeEach(func() {
			client = new(mocks.Client)
			asMock = client.(*mocks.Client)

			fc = commands.FunctionCreate(&client)
		})
		AfterEach(func() {
			asMock.AssertExpectations(GinkgoT())

		})
		It("should involve the core.Client", func() {
			fc.SetArgs([]string{"node", "square", "--image", "foo/bar", "--git-repo", "https://github.com/repo"})

			o := core.CreateFunctionOptions{
				GitRepo:     "https://github.com/repo",
				GitRevision: "master",
				InvokerURL:  "https://github.com/projectriff/node-function-invoker/raw/v0.0.8/node-invoker.yaml",
			}
			o.Name = "square"
			o.Image = "foo/bar"
			o.Env = []string{}
			o.EnvFrom = []string{}

			asMock.On("CreateFunction", o).Return(nil, nil)
			err := fc.Execute()
			Expect(err).NotTo(HaveOccurred())
		})
		It("should propagate core.Client errors", func() {
			fc.SetArgs([]string{"node", "square", "--image", "foo/bar", "--git-repo", "https://github.com/repo"})

			e := fmt.Errorf("some error")
			asMock.On("CreateFunction", mock.Anything).Return(nil, e)
			err := fc.Execute()
			Expect(err).To(MatchError(e))
		})
		It("should add env vars when asked to", func() {
			fc.SetArgs([]string{"node", "square", "--image", "foo/bar", "--git-repo", "https://github.com/repo",
				"--env", "FOO=bar", "--env", "BAZ=qux", "--env-from", "secretKeyRef:foo:bar"})

			o := core.CreateFunctionOptions{
				GitRepo:     "https://github.com/repo",
				GitRevision: "master",
				InvokerURL:  "https://github.com/projectriff/node-function-invoker/raw/v0.0.8/node-invoker.yaml",
			}
			o.Name = "square"
			o.Image = "foo/bar"
			o.Env = []string{"FOO=bar", "BAZ=qux"}
			o.EnvFrom = []string{"secretKeyRef:foo:bar"}

			asMock.On("CreateFunction", o).Return(nil, nil)
			err := fc.Execute()
			Expect(err).NotTo(HaveOccurred())
		})
		It("should create channel/subscription when asked to", func() {
			fc.SetArgs([]string{"node", "square", "--image", "foo/bar", "--git-repo", "https://github.com/repo",
				"--input", "my-channel", "--bus", "kafka"})

			functionOptions := core.CreateFunctionOptions{
				GitRepo:     "https://github.com/repo",
				GitRevision: "master",
				InvokerURL:  "https://github.com/projectriff/node-function-invoker/raw/v0.0.8/node-invoker.yaml",
			}
			functionOptions.Name = "square"
			functionOptions.Image = "foo/bar"
			functionOptions.Env = []string{}
			functionOptions.EnvFrom = []string{}

			channelOptions := core.CreateChannelOptions{
				Name: "my-channel",
				Bus:  "kafka",
			}
			subscriptionOptions := core.CreateSubscriptionOptions{
				Name:       "square",
				Channel:    "my-channel",
				Subscriber: "square",
			}

			asMock.On("CreateFunction", functionOptions).Return(nil, nil)
			asMock.On("CreateChannel", channelOptions).Return(nil, nil)
			asMock.On("CreateSubscription", subscriptionOptions).Return(nil, nil)
			err := fc.Execute()
			Expect(err).NotTo(HaveOccurred())
		})
		It("should print when --dry-run is set", func() {
			fc.SetArgs([]string{"node", "square", "--image", "foo/bar", "--git-repo", "https://github.com/repo",
				"--input", "my-channel", "--bus", "kafka", "--dry-run"})

			functionOptions := core.CreateFunctionOptions{
				GitRepo:     "https://github.com/repo",
				GitRevision: "master",
				InvokerURL:  "https://github.com/projectriff/node-function-invoker/raw/v0.0.8/node-invoker.yaml",
			}
			functionOptions.Name = "square"
			functionOptions.Image = "foo/bar"
			functionOptions.Env = []string{}
			functionOptions.EnvFrom = []string{}
			functionOptions.DryRun = true

			channelOptions := core.CreateChannelOptions{
				Name:   "my-channel",
				Bus:    "kafka",
				DryRun: true,
			}
			subscriptionOptions := core.CreateSubscriptionOptions{
				Name:       "square",
				Channel:    "my-channel",
				Subscriber: "square",
				DryRun:     true,
			}

			f := v1alpha1.Service{}
			f.Name = "square"
			c := v1alpha12.Channel{}
			c.Name = "my-channel"
			s := v1alpha12.Subscription{}
			s.Name = "square"
			asMock.On("CreateFunction", functionOptions).Return(&f, nil)
			asMock.On("CreateChannel", channelOptions).Return(&c, nil)
			asMock.On("CreateSubscription", subscriptionOptions).Return(&s, nil)

			stdout := &strings.Builder{}
			fc.SetOutput(stdout)

			err := fc.Execute()
			Expect(err).NotTo(HaveOccurred())

			Expect(stdout.String()).To(Equal(fnCreateDryRun))
		})

	})
})

const fnCreateDryRun = `metadata:
  creationTimestamp: null
  name: square
spec: {}
status: {}
---
metadata:
  creationTimestamp: null
  name: my-channel
spec: {}
status: {}
---
metadata:
  creationTimestamp: null
  name: square
spec:
  channel: ""
  subscriber: ""
status: {}
---
`
