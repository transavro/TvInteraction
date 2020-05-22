package main

import (
	"golang.org/x/net/context"
	"os/exec"
)

type Server struct {}

func (s Server) KeycodeAction(_ context.Context, req *KeycodeReq) (*Resp, error) {
	return &Resp{}, exec.Command("am",
		"broadcast", "-a",
		"tv.cloudwalker.interaction.action.KEYCODE",
		"--ei", "keycode",
		req.Keycode).Run()
}

func (s Server) GetAllApps(_ context.Context, _ *GetAllAppsReq) (*GetAllAppsResp, error) {
	if result, err := getInstalledAppList(); err != nil {
		return nil, err
	} else {
		return &GetAllAppsResp{AppList: result}, nil
	}
}

func (s Server) AppAction(_ context.Context, req *AppReq) (*Resp, error) {
	switch req.App {
	case App_APP_OPEN:
		{
			return &Resp{}, exec.Command("monkey", "-p", req.Package, "1").Run()
		}
	case App_APP_UNINSTALL:
		{
			return &Resp{}, exec.Command("pm uninstall", req.Package).Run()
		}
	default:
		return nil, nil
	}
}

func (s Server) PlayContent(_ context.Context, req *PlayReq) (*Resp, error) {
	return &Resp{}, exec.Command("am", "start", "-a", "android.intent.action.VIEW", "-d", req.DeepLink, req.Package).Run()
}

func (s Server) GetListOfTvSources(_ context.Context, _ *Req) (*TvSources, error) {
	return &TvSources{Tvsources: getTvSources()}, nil
}

func (s Server) TvAction(_ context.Context, req *TvActionReq) (*Resp, error) {
	return &Resp{}, exec.Command("am",
		"broadcast", "-a",
		"tv.cloudwalker.interaction.action.SOURCE",
		"--es", "tvsource",
		req.Source).Run()
}
