package vehicle

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/vw"
	"golang.org/x/oauth2"
)

// https://github.com/davidgiga1993/AudiAPI
// https://github.com/TA2k/ioBroker.vw-connect

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Audi{
		embed: &cc.embed,
	}

	log := util.NewLogger("audi")
	identity := vw.NewIdentity(log)

	query := url.Values(map[string][]string{
		"response_type": {"id_token token"},
		"client_id":     {"09b6cbec-cd19-4589-82fd-363dfa8c24da@apps_vw-dilab_com"},
		"redirect_uri":  {"myaudi:///"},
		"scope":         {"openid profile mbb vin badge birthdate nickname email address phone name picture"},
		"prompt":        {"login"},
		"ui_locales":    {"de-DE"},
	})

	err := identity.LoginVAG("77869e21-e30a-4a92-b016-48ab7d3db1d8", query, cc.User, cc.Password)
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, "Audi", "DE")
	api.Client.Timeout = cc.Timeout

	if cc.VIN == "" {
		cc.VIN, err = findVehicle(api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", cc.VIN)
		}
	}

	if err == nil {
		if err = api.HomeRegion(strings.ToUpper(cc.VIN)); err == nil {
			v.Provider = vw.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)
		}
	}

	return v, err
}

type AudiVehicles struct {
	Vehicles []AudiVehicle
}

type AudiVehicle struct {
	VIN       string
	ShortName string
	ImageUrl  string
}

func (v *Audi) Images(identity *vw.Identity, vin string) ([]string, error) {
	helper := request.NewHelper(util.NewLogger("IMAGE"))
	helper.Client.Transport = &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: identity.AccessToken(),
		}),
		Base: helper.Client.Transport,
	}

	uri := "https://api.my.audi.com/smns/v1/navigation/v1/vehicles"
	// uri := fmt.Sprintf("https://api.my.audi.com/smns/v1/navigation/v1/vehicles/%s", vin)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"X-Market": "de_DE",
	})

	var res AudiVehicles
	if err == nil {
		err = helper.DoJSON(req, &res)
		fmt.Println(res)
	}

	panic(err)
}
