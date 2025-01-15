package db

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/juanfont/headscale/hscontrol/types"
	"github.com/juanfont/headscale/hscontrol/util"
	"gorm.io/gorm"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

func (hsdb *HSDatabase) RegisterWireguardOnlyNode(
	nkey key.NodePublic,
	mkey key.MachinePublic,
	userID types.UserID,
	ipv4 *netip.Addr,
	ipv6 *netip.Addr,
	nodeIpv4 *netip.Addr,
	nodeIpv6 *netip.Addr,
	req *v1.RegisterWireguardPeerRequest,
) (*types.Node, error) {
	written, err := Write(hsdb.DB, func(tx *gorm.DB) (*types.Node, error) {
		now := time.Now().UTC()
		location := &tailcfg.Location{
			Country:     req.Country,
			CountryCode: req.CountryCode,
			City:        req.City,
			CityCode:    req.CityCode,
			Latitude:    float64(req.Latitude),
			Longitude:   float64(req.Longitude),
		}
		user, err := GetUserByID(tx, userID)
		if err != nil {
			return nil, fmt.Errorf("looking up user: %w", err)
		}
		discrim, err := util.GenerateRandomStringDNSSafe(10)
		hostname := strings.ToLower(fmt.Sprintf("%s-%s-%s", req.CountryCode, req.CityCode, discrim))
		if err != nil {
			return nil, err
		}
		n := &types.Node{
			UserID:         user.ID,
			User:           *user,
			RegisterMethod: util.RegisterMethodCLI,
			Hostinfo: &tailcfg.Hostinfo{
				RoutableIPs: []netip.Prefix{
					netip.MustParsePrefix("0.0.0.0/0"),
					netip.MustParsePrefix("::/0"),
					netip.PrefixFrom(*ipv4, 32),
					netip.PrefixFrom(*ipv6, 128),
				},
				Location: location,
			},
			GivenName:       hostname,
			Hostname:        hostname,
			NodeKey:         nkey,
			LastSeen:        &now,
			IsWireguardOnly: true,
			IPv4:            ipv4,
			IPv6:            ipv6,
			Endpoints: []netip.AddrPort{
				netip.AddrPortFrom(*ipv4, uint16(req.Port)),
				netip.AddrPortFrom(*ipv6, uint16(req.Port)),
			},
			Location: location,
		}
		err = tx.Save(n).Error
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return nil, err
	}
	_, err = hsdb.SaveNodeRoutes(written)
	return written, err
}

func (hsdb *HSDatabase) ListWgPeers() (types.Nodes, error) {
	return Read(hsdb.DB, func(rx *gorm.DB) (types.Nodes, error) {
		return ListWgPeers(rx)
	})
}

func ListWgPeers(tx *gorm.DB) (types.Nodes, error) {
	nodes := types.Nodes{}
	if err := tx.
		Preload("AuthKey").
		Preload("AuthKey.User").
		Preload("User").
		Preload("Routes").
		Where("is_wireguard_only = ?", true).
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	return nodes, nil
}
