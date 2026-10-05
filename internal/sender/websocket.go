package sender

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	wsegress "github.com/chapar-rest/chapar/internal/egress/websocket"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/variables"
	"github.com/chapar-rest/chapar/version"
)

// ConnectWebSocket runs the request's pre-request step, then opens its
// connection. The returned response describes the handshake: status,
// headers, cookies, timeline and script tests. It is returned on failure
// too, when there is anything to show.
//
// onEvent gets every event of the connection, from other goroutines.
// Post-request actions do not apply: a connection has no single response.
func (s *Service) ConnectWebSocket(ctx context.Context, req *domain.Request, env *domain.Environment, onEvent func(wsegress.Event)) (*wsegress.Session, *egress.Response, error) {
	if req == nil || req.Spec.WebSocket == nil {
		return nil, nil, fmt.Errorf("invalid websocket request")
	}

	var collection *domain.Collection
	if req.CollectionID != "" && s.colls != nil {
		collection = s.colls(req.CollectionID)
	}

	var timeline []egress.TimelineStep
	preStep, preResult, err := s.preRequestTimed(req, env, collection)
	if preStep != nil {
		timeline = append(timeline, *preStep)
	}
	tests := scriptTests(preResult)
	if err != nil {
		return nil, &egress.Response{Timeline: timeline, Error: err, ScriptTests: tests}, err
	}
	if preResult != nil && preResult.Skip {
		err := errors.New("the pre-request script skipped this connection")
		if preResult.SkipReason != "" {
			err = fmt.Errorf("the pre-request script skipped this connection: %s", preResult.SkipReason)
		}
		return nil, &egress.Response{Timeline: timeline, Error: err, ScriptTests: tests}, err
	}

	r := req.Clone()
	r.MetaData.ID = req.MetaData.ID
	if preResult != nil && !preResult.Request.Empty() {
		scripting.ApplyChanges(r, preResult.Request)
		if preResult.Request.Headers != nil {
			// The script saw the collection headers merged in.
			collection = nil
		}
	}
	spec := r.Spec.WebSocket
	if collection != nil {
		spec.Headers = domain.MergeHeaders(collection.Spec.Headers, spec.Headers)
		if spec.Auth.Type == domain.AuthTypeInherit {
			spec.Auth = collection.Spec.Auth.Clone()
		}
	}

	vars := variables.GetVariables()
	variables.ApplyToWebSocketRequest(vars, spec)
	if e := copyEnv(env); e != nil {
		variables.ApplyToEnv(vars, &e.Spec)
		e.ApplyToWebSocketRequest(spec)
	}

	envID := ""
	if env != nil {
		envID = env.ID()
	}
	session, hs, err := wsegress.Dial(ctx, spec, s.webSocketOptions(spec, envID), onEvent)
	s.saveCookies(env)

	res := &egress.Response{Timeline: timeline, ScriptTests: tests, Error: err}
	if hs != nil {
		res.StatusCode = hs.StatusCode
		res.RequestHeaders = hs.RequestHeaders
		res.ResponseHeaders = hs.ResponseHeaders
		res.Cookies = hs.Cookies
		res.CookieEvents = hs.CookieEvents
		res.SentCookies = hs.SentCookies
		res.TimePassed = hs.Duration
		res.Timeline = append(res.Timeline, hs.Timeline...)
	}
	if err != nil {
		return nil, res, err
	}
	return session, res, nil
}

// ResolveText fills dynamic variables and the environment's values into a
// message about to be sent.
func (s *Service) ResolveText(text string, env *domain.Environment) string {
	vars := variables.GetVariables()
	text = variables.ApplyToString(vars, text)
	if e := copyEnv(env); e != nil {
		variables.ApplyToEnv(vars, &e.Spec)
		text = e.ApplyToString(text)
	}
	return text
}

func (s *Service) webSocketOptions(spec *domain.WebSocketRequestSpec, envID string) wsegress.Options {
	general := prefs.GetGlobalConfig().Spec.General
	opts := wsegress.Options{
		ConnectTimeout:     time.Duration(general.RequestTimeoutSec) * time.Second,
		PingInterval:       time.Duration(spec.Settings.PingIntervalSec) * time.Second,
		ReadLimit:          int64(general.ResponseSizeMb) << 20,
		InsecureSkipVerify: !general.VaidateTLSCertificates,
		Compression:        spec.Settings.Compression,
		Cookies:            s.cookies,
		EnvID:              envID,
	}
	if spec.Settings.ConnectTimeoutSec > 0 {
		opts.ConnectTimeout = time.Duration(spec.Settings.ConnectTimeoutSec) * time.Second
	}
	if general.SendChaparAgentHeader {
		opts.UserAgent = version.GetAgentName()
	}
	return opts
}
