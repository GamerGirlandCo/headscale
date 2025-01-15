package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(peerCmd)
	registerPeerCmd.Flags().String("user", "", "User name")
	registerPeerCmd.Flags().StringP("cfg", "f", "", "Config file")
	peerCmd.AddCommand(registerPeerCmd)
}

var peerCmd = &cobra.Command{
	Use:     "peers",
	Short:   "Manage the wireguard-only peers of Headscale",
	Aliases: []string{"peer"},
}

type rawPeerConfig struct {
	PublicKey   string `yaml:"public_key"`
	Ip4         string `yaml:"ipv4"`
	Ip6         string `yaml:"ipv6"`
	Country     string `yaml:"country"`
	CountryCode string `yaml:"country_code"`
	City        string `yaml:"city"`
	CityCode    string `yaml:"city_code"`
	Latitude    string `yaml:"latitude"`
	Longitude   string `yaml:"longitude"`
	Port        string `yaml:"port"`
}

var registerPeerCmd = &cobra.Command{
	Use:   "register",
	Short: "Registers a wireguard-only peer to your network",
	Run: func(cmd *cobra.Command, args []string) {

		output, _ := cmd.Flags().GetString("output")
		ctx, client, conn, cancel := newHeadscaleCLIWithConfig()
		defer cancel()
		defer conn.Close()

		user, err := cmd.Flags().GetString("user")
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Error getting user: %s\n", err),
				output,
			)
		}
		conf, err := cmd.Flags().GetString("cfg")
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Error getting peer config: %s\n", err),
				output,
			)
			return
		}
		var rpc rawPeerConfig
		var bytes []byte

		if conf == "-" {
			bytes, err = io.ReadAll(os.Stdin)	
		} else {
			bytes, err = os.ReadFile(conf)
		}
		if err != nil {
				ErrorOutput(
					err,
					fmt.Sprintf("Error reading peer config: %s\n", err),
					output,
				)
				return
			}
		err = yaml.Unmarshal(bytes, &rpc)
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf("Error parsing peer config: %s\n", err),
				output,
			)
		}

		port, err := strconv.Atoi(rpc.Port)
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf(
					"Cannot register peer: %s\n",
					status.Convert(err).Message(),
				),
				output,
			)
			return
		}
		lat, err := strconv.ParseFloat(rpc.Latitude, 32)
		if err != nil {
			lat = 0
		}
		long, err := strconv.ParseFloat(rpc.Longitude, 32)
		if err != nil {
			long = 0
		}
		res, err := client.RegisterPeer(ctx, &v1.RegisterWireguardPeerRequest{
			User:        user,
			PubKey:      rpc.PublicKey,
			Ipv4:        rpc.Ip4,
			Ipv6:        rpc.Ip6,
			Port:        uint32(port),
			Country:     rpc.Country,
			CountryCode: rpc.CountryCode,
			Longitude:   long,
			Latitude:    lat,
			City:        rpc.City,
			CityCode:    rpc.CityCode,
		})
		if err != nil {
			ErrorOutput(
				err,
				fmt.Sprintf(
					"Cannot register peer: %s\n",
					status.Convert(err).Message(),
				),
				output,
			)
			return
		}
		SuccessOutput(
			res.GetNode(),
			fmt.Sprintf("Peer registered at [%s] ([%s])", rpc.Ip4, rpc.Ip6),
			output)
	},
}
