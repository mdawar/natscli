// Copyright 2020-2022 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/choria-io/fisk"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

type SrvRequestCmd struct {
	name    string
	host    string
	cluster string
	account string
	tags    []string

	limit   int
	offset  int
	waitFor uint32

	includeAccounts  bool
	includeStreams   bool
	includeConsumers bool
	includeConfig    bool
	leaderOnly       bool
	includeAll       bool

	detail        bool
	sortOpt       string
	cidFilter     uint64
	stateFilter   string
	userFilter    string
	accountFilter string
	subjectFilter string
	nameFilter    string

	jsServerOnly bool
	jsEnabled    bool

	profileName  string
	profileDebug int
	profileDir   string
	filterEmpty  bool
}

func configureServerRequestCommand(srv *fisk.CmdClause) {
	c := &SrvRequestCmd{}

	req := srv.Command("request", "Request monitoring data from a specific server").Alias("req")
	req.Flag("limit", "Limit the responses to a certain amount of records").Default("2048").IntVar(&c.limit)
	req.Flag("offset", "Start at a certain record").Default("0").IntVar(&c.offset)
	req.Flag("name", "Limit to servers matching a server name").StringVar(&c.name)
	req.Flag("host", "Limit to servers matching a server host name").StringVar(&c.host)
	req.Flag("cluster", "Limit to servers matching a cluster name").StringVar(&c.cluster)
	req.Flag("tags", "Limit to servers with these configured tags").StringsVar(&c.tags)

	subz := req.Command("subscriptions", "Show subscription information").Alias("sub").Alias("subsz").Action(c.subs)
	subz.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)
	subz.Flag("detail", "Include detail about all subscriptions").UnNegatableBoolVar(&c.detail)
	subz.Flag("filter-account", "Filter on a specific account").PlaceHolder("ACCOUNT").StringVar(&c.accountFilter)
	subz.Flag("filter-subject", "Filter based on subscriptions matching this subject").PlaceHolder("SUBJECT").StringVar(&c.subjectFilter)

	varz := req.Command("variables", "Show runtime variables").Alias("var").Alias("varz").Action(c.varz)
	varz.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)

	connz := req.Command("connections", "Show connection details").Alias("conn").Alias("connz").Action(c.conns)
	connz.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)
	connz.Flag("sort", "Sort by a specific property").Default("cid").EnumVar(&c.sortOpt, "cid", "start", "subs", "pending", "msgs_to", "msgs_from", "bytes_to", "bytes_from", "last", "idle", "uptime", "stop", "reason", "rtt")
	connz.Flag("subscriptions", "Show subscriptions").UnNegatableBoolVar(&c.detail)
	connz.Flag("filter-cid", "Filter on a specific CID").PlaceHolder("CID").Uint64Var(&c.cidFilter)
	connz.Flag("filter-state", "Filter on a specific account state (open, closed, all)").PlaceHolder("STATE").Default("open").EnumVar(&c.stateFilter, "open", "closed", "all")
	connz.Flag("filter-user", "Filter on a specific username").PlaceHolder("USER").StringVar(&c.userFilter)
	connz.Flag("filter-account", "Filter on a specific account").PlaceHolder("ACCOUNT").StringVar(&c.accountFilter)
	connz.Flag("filter-subject", "Limits responses only to those connections with matching subscription interest").PlaceHolder("SUBJECT").StringVar(&c.subjectFilter)
	connz.Flag("filter-empty", "Only shows responses that have connections").Default("false").UnNegatableBoolVar(&c.filterEmpty)

	routez := req.Command("routes", "Show route details").Alias("route").Alias("routez").Action(c.routez)
	routez.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)
	routez.Flag("subscriptions", "Show subscription detail").UnNegatableBoolVar(&c.detail)

	gwyz := req.Command("gateways", "Show gateway details").Alias("gateway").Alias("gwy").Alias("gatewayz").Action(c.gwyz)
	gwyz.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)
	gwyz.Arg("filter-name", "Filter results on gateway name").PlaceHolder("NAME").StringVar(&c.nameFilter)
	gwyz.Flag("filter-account", "Show only a certain account in account detail").PlaceHolder("ACCOUNT").StringVar(&c.accountFilter)
	gwyz.Flag("accounts", "Show account detail").UnNegatableBoolVar(&c.detail)

	leafz := req.Command("leafnodes", "Show leafnode details").Alias("leaf").Alias("leafz").Action(c.leafz)
	leafz.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)
	leafz.Flag("subscriptions", "Show subscription detail").UnNegatableBoolVar(&c.detail)

	accountz := req.Command("accounts", "Show account details").Alias("accountz").Alias("acct").Action(c.accountz)
	accountz.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)
	accountz.Flag("account", "Retrieve information for a specific account").StringVar(&c.account)

	jsz := req.Command("jetstream", "Show JetStream details").Alias("jsz").Alias("js").Action(c.jsz)
	jsz.Arg("wait", "Wait for a certain number of responses").Uint32Var(&c.waitFor)
	jsz.Flag("account", "Show statistics scoped to a specific account").StringVar(&c.account)
	jsz.Flag("accounts", "Include details about accounts").UnNegatableBoolVar(&c.includeAccounts)
	jsz.Flag("streams", "Include details about Streams").UnNegatableBoolVar(&c.includeStreams)
	jsz.Flag("consumer", "Include details about Consumers").UnNegatableBoolVar(&c.includeConsumers)
	jsz.Flag("config", "Include details about configuration").UnNegatableBoolVar(&c.includeConfig)
	jsz.Flag("leader", "Request a response from the Meta-group leader only").UnNegatableBoolVar(&c.leaderOnly)
	jsz.Flag("all", "Include accounts, streams, consumers and configuration").UnNegatableBoolVar(&c.includeAll)

	healthz := req.Command("jetstream-health", "Request JetStream health status").Alias("healthz").Action(c.healthz)
	healthz.Flag("js-enabled", "Checks that JetStream should be enabled on all servers").Short('J').BoolVar(&c.jsEnabled)
	healthz.Flag("server-only", "Restricts the health check to the JetStream server only, do not check streams and consumers").Short('S').BoolVar(&c.jsServerOnly)

	profilez := req.Command("profile", "Run a profile").Action(c.profilez)
	profilez.Arg("profile", "Specify the name of the profile to run (allocs, heap, goroutine, mutex, threadcreate, block)").StringVar(&c.profileName)
	profilez.Arg("dir", "Set the output directory for profile files").Default(".").ExistingDirVar(&c.profileDir)
	profilez.Flag("debug", "Set the debug level of the profile").IntVar(&c.profileDebug)
}

func (c *SrvRequestCmd) healthz(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.HealthzEventOptions{
		HealthzOptions:     server.HealthzOptions{JSEnabledOnly: c.jsEnabled, JSServerOnly: c.jsServerOnly},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("HEALTHZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

type profilezResponse struct {
	Server server.ServerInfo     `json:"server"`
	Resp   server.ProfilezStatus `json:"data"`
}

func (c *SrvRequestCmd) profilez(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.ProfilezEventOptions{
		ProfilezOptions: server.ProfilezOptions{
			Name:  c.profileName,
			Debug: c.profileDebug,
		},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("PROFILEZ", &opts, nc)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("%s-%s-", c.profileName, time.Now().Format("20060102-150405"))
	prefix = filepath.Join(c.profileDir, prefix)

	for _, r := range res {
		var resp profilezResponse
		if err := json.Unmarshal(r, &resp); err != nil {
			continue
		}
		if resp.Resp.Error != "" {
			fmt.Fprintf(os.Stderr, "Server %q error: %s\n", resp.Server.Name, resp.Resp.Error)
			continue
		}

		filename := prefix + resp.Server.Name
		if err := c.profilezWrite(filename, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Server %q error: %s\n", resp.Server.Name, err)
		} else {
			fmt.Fprintf(os.Stdout, "Server %q profile written: %s\n", resp.Server.Name, filename)
		}
	}

	return nil
}

func (c *SrvRequestCmd) profilezWrite(filename string, resp *profilezResponse) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	n, err := f.Write(resp.Resp.Profile)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if n != len(resp.Resp.Profile) {
		return fmt.Errorf("short write")
	}

	return nil
}

func (c *SrvRequestCmd) jsz(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.JszEventOptions{
		JSzOptions:         server.JSzOptions{Account: c.account, LeaderOnly: c.leaderOnly},
		EventFilterOptions: c.reqFilter(),
	}

	if c.includeAccounts || c.includeAll {
		opts.JSzOptions.Accounts = true
	}
	if c.includeStreams || c.includeAll {
		opts.JSzOptions.Streams = true
	}
	if c.includeConsumers || c.includeAll {
		opts.JSzOptions.Consumer = true
	}
	if c.includeConfig || c.includeAll {
		opts.JSzOptions.Config = true
	}

	res, err := c.doReq("JSZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) reqFilter() server.EventFilterOptions {
	opt := server.EventFilterOptions{
		Name:    c.name,
		Host:    c.host,
		Cluster: c.cluster,
		Tags:    c.tags,
	}
	if opts.Config != nil {
		opt.Domain = opts.Config.JSDomain()
	}

	return opt
}

func (c *SrvRequestCmd) accountz(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.AccountzEventOptions{
		AccountzOptions:    server.AccountzOptions{Account: c.account},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("ACCOUNTZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) leafz(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.LeafzEventOptions{
		LeafzOptions:       server.LeafzOptions{Subscriptions: c.detail},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("LEAFZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) gwyz(_ *fisk.ParseContext) error {
	if c.accountFilter != "" {
		c.detail = true
	}

	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.GatewayzEventOptions{
		GatewayzOptions: server.GatewayzOptions{
			Name:        c.nameFilter,
			Accounts:    c.detail,
			AccountName: c.accountFilter,
		},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("GATEWAYZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) routez(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.RoutezEventOptions{
		RoutezOptions: server.RoutezOptions{
			Subscriptions:       c.detail,
			SubscriptionsDetail: c.detail,
		},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("ROUTEZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) conns(_ *fisk.ParseContext) error {
	opts := &server.ConnzEventOptions{
		ConnzOptions: server.ConnzOptions{
			Sort:                server.SortOpt(c.sortOpt),
			Username:            true,
			Subscriptions:       c.detail,
			SubscriptionsDetail: c.detail,
			Offset:              c.offset,
			Limit:               c.limit,
			CID:                 c.cidFilter,
			User:                c.userFilter,
			Account:             c.accountFilter,
			FilterSubject:       c.subjectFilter,
		},
		EventFilterOptions: c.reqFilter(),
	}

	switch c.stateFilter {
	case "open":
		opts.State = 0
	case "closed":
		opts.State = 1
	default:
		opts.State = 2
	}

	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	res, err := c.doReq("CONNZ", opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		if c.filterEmpty {
			var r server.ServerAPIConnzResponse
			err = json.Unmarshal(m, &r)
			if err == nil {
				if r.Data == nil || r.Data.NumConns == 0 {
					continue
				}
			}
		}

		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) varz(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.VarzEventOptions{
		VarzOptions:        server.VarzOptions{},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("VARZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) subs(_ *fisk.ParseContext) error {
	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	opts := server.SubszEventOptions{
		SubszOptions: server.SubszOptions{
			Offset:        c.offset,
			Limit:         c.limit,
			Subscriptions: c.detail,
			Account:       c.accountFilter,
			Test:          c.subjectFilter,
		},
		EventFilterOptions: c.reqFilter(),
	}

	res, err := c.doReq("SUBSZ", &opts, nc)
	if err != nil {
		return err
	}

	for _, m := range res {
		fmt.Println(string(m))
	}

	return nil
}

func (c *SrvRequestCmd) doReq(kind string, req any, nc *nats.Conn) ([][]byte, error) {
	return doReq(req, fmt.Sprintf("$SYS.REQ.SERVER.PING.%s", kind), int(c.waitFor), nc)
}
