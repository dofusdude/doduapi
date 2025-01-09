package config

import (
	"time"

	"github.com/dofusdude/ankabuffer"
	"github.com/dofusdude/doduapi/utils"
)

var (
	Languages               = []string{"de", "en", "es", "fr", "pt"}
	ItemImgResolutions      = []string{"64", "128"}
	MountImgResolutions     = []string{"64", "256"}
	ApiHostName             string
	ApiPort                 string
	ApiScheme               string
	DockerMountDataPath     string
	MajorVersion            int
	AlmanaxMaxLookAhead     int
	AlmanaxDefaultLookAhead int
	DbDir                   string
	FileHashes              ankabuffer.Manifest // TODO why is this here?
	MeiliHost               string
	MeiliKey                string
	PrometheusEnabled       bool
	PublishFileServer       bool
	PersistedElements       utils.PersistentStringKeysMap // TODO remove, since not a fixed config param
	PersistedTypes          utils.PersistentStringKeysMap // TODO remove, since not a fixed config param
	IsBeta                  bool
	LastUpdate              time.Time // TODO remove, since not a fixed config param
	ElementsUrl             string
	TypesUrl                string
	ReleaseUrl              string
	UpdateHookToken         string
	DofusVersion            string
	CurrentVersion          utils.GameVersion // TODO remove, since not a fixed config param
	ApiVersion              string
)
