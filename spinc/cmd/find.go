// Copyright 2020, Square, Inc.

package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/square/spincycle/v2/proto"
	"github.com/square/spincycle/v2/spinc/app"
)

const (
	findLimitDefault = 10 // limit to 10 requests by default

	// formatting for outputing request info
	findReqColLen   = 40
	findIdColLen    = 20
	findUserColLen  = 9
	findStateColLen = 9
	findJobsColLen  = 15

	findTimeFmt    = "YYYY-MM-DD HH:MM:SS UTC" // expected time input format
	findTimeFmtStr = "2006-01-02 15:04:05 MST" // expected time input format as the actual format string (input to time.Parse)
)

var (
	findTimeColLen = len(findTimeFmt)
	findUtcIndex   = strings.Index(findTimeFmt, "UTC")
)

type Find struct {
	ctx    app.Context
	filter proto.RequestFilter
}

func NewFind(ctx app.Context) *Find {
	return &Find{
		ctx: ctx,
	}
}

func (c *Find) Prepare() error {
	/* Parse. */
	// See command usage for details about each filter
	validFilters := map[string]bool{
		"type":   true,
		"states": true,
		"user":   true,
		"since":  true,
		"until":  true,
		"limit":  true,
		"offset": true,
	}
	filters := map[string]string{}
	for _, arg := range c.ctx.Command.Args {
		split := strings.SplitN(arg, "=", 2)
		if len(split) != 2 {
			return fmt.Errorf("Invalid command arg: %s: split on = produced %d values, expected 2 (filter=value)", arg, len(split))
		}
		filter := split[0]
		value := split[1]

		if !validFilters[filter] {
			return fmt.Errorf("Invalid filter '%s'", filter)
		}
		if _, ok := filters[filter]; ok {
			return fmt.Errorf("Filter '%s' specified multiple times", filter)
		}
		filters[filter] = value

		if c.ctx.Options.Debug {
			app.Debug("filter '%s'='%s'", filter, value)
		}
	}

	/* Process some args. */
	var err error

	states := []byte{}
	if len(filters["states"]) > 0 {
		for _, state := range strings.Split(filters["states"], ",") {
			val, ok := proto.StateValue[state]
			if !ok {
				expected := make([]string, 0, len(proto.StateValue))
				for k, _ := range proto.StateValue {
					expected = append(expected, k)
				}
				return fmt.Errorf("Invalid state '%s', expected one of: %s", state, strings.Join(expected, ", "))
			}
			states = append(states, val)
		}
	}

	var since time.Time
	if filters["since"] != "" {
		if strings.Index(filters["since"], "UTC") != findUtcIndex {
			return fmt.Errorf("Invalid time %s, expected string 'UTC' at index %d", filters["since"], findUtcIndex)
		}
		since, err = time.Parse(findTimeFmtStr, filters["since"])
		if err != nil {
			return fmt.Errorf("Invalid time %s, expected form '%s'", filters["since"], findTimeFmt)
		}
	}

	var until time.Time
	if filters["until"] != "" {
		if strings.Index(filters["until"], "UTC") != findUtcIndex {
			return fmt.Errorf("Invalid time %s, expected string 'UTC' at index %d", filters["until"], findUtcIndex)
		}
		until, err = time.Parse(findTimeFmtStr, filters["until"])
		if err != nil {
			return fmt.Errorf("Invalid time %s, expected form '%s'", filters["until"], findTimeFmt)
		}
	}

	var limit uint
	if filters["limit"] == "" {
		limit = findLimitDefault
	} else {
		l, err := strconv.ParseUint(filters["limit"], 10, strconv.IntSize)
		if err != nil {
			return fmt.Errorf("Invalid limit '%s', expected value >= 0", filters["limit"])
		}
		limit = uint(l)
	}

	var offset uint
	if filters["offset"] != "" {
		o, err := strconv.ParseUint(filters["offset"], 10, strconv.IntSize)
		if err != nil {
			return fmt.Errorf("Invalid offset '%s', expected value >= 0", filters["offset"])
		}
		offset = uint(o)
	}

	/* Build the request filter. */
	c.filter = proto.RequestFilter{
		Type:   filters["type"],
		States: states,
		User:   filters["user"],

		Since: since,
		Until: until,

		Limit:  limit,
		Offset: offset,
	}

	return nil
}

func (c *Find) Run() error {
	requests, err := c.ctx.RMClient.FindRequests(c.filter)
	if err != nil {
		return err
	}
	if c.ctx.Options.Debug {
		app.Debug("requests: %#v", requests)
	}

	if c.ctx.Hooks.CommandRunResult != nil {
		c.ctx.Hooks.CommandRunResult(requests, err)
		return nil
	}

	if len(requests) == 0 {
		return nil
	}

	/*
	   ID                   REQUEST                                  USER      STATE     CREATED STARTED FINISHED JOBS            HOST
	   -------------------- 1234567890123456789012345678901234567890 123456789 123456789 ------- ------- -------- 123456789012345 *
	*/
	line := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds %%s\n",
		findIdColLen, findReqColLen, findUserColLen, findStateColLen, findTimeColLen, findTimeColLen, findTimeColLen, findJobsColLen)

	fmt.Fprintf(c.ctx.Out, line, "ID", "REQUEST", "USER", "STATE", "CREATED", "STARTED", "FINISHED", "JOBS", "HOST")

	for _, r := range requests {
		state, ok := proto.StateName[r.State]
		if !ok {
			state = proto.StateName[proto.STATE_UNKNOWN]
		}

		createdAt := r.CreatedAt.Local().Format(findTimeFmtStr)

		startedAt := "N/A"
		if r.StartedAt != nil {
			startedAt = r.StartedAt.Local().Format(findTimeFmtStr)
		}

		finishedAt := "N/A"
		if r.FinishedAt != nil {
			finishedAt = r.FinishedAt.Local().Format(findTimeFmtStr)
		}

		jobs := fmt.Sprintf("%d / %d", r.FinishedJobs, r.TotalJobs)

		fmt.Fprintf(c.ctx.Out, line,
			SqueezeString(r.Id, findIdColLen, ".."),
			SqueezeString(r.Type, findReqColLen, ".."),
			SqueezeString(r.User, findUserColLen, ".."),
			SqueezeString(state, findStateColLen, ".."),
			createdAt, startedAt, finishedAt,
			SqueezeString(jobs, findJobsColLen, ".."),
			r.JobRunnerURL)
	}

	return nil
}

func (c *Find) Cmd() string {
	if len(c.ctx.Command.Args) > 0 {
		return "find " + strings.Join(c.ctx.Command.Args, " ")
	}
	return "find"
}

func (c *Find) Help() string {
	return fmt.Sprintf(`'spinc find [filter=value]' retrieves and filters requests.

Filters:
  type        type of request to return
  states      comma-separated list of request states to include
  user        return only requests made by this user
  since       return requests created or run after this time
  until       return requests created or run before this time
  limit       limit response to this many requests (default: %d)
  offset      skip the first <offset> requests

Times should be formated as '%s'. Time should be specified in UTC.
`, findLimitDefault, findTimeFmt)
}
