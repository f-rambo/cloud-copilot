package utils

import (
	"fmt"
	"math"
	"math/big"
	"net"

	"github.com/pkg/errors"
)

func GenerateCIDR(ip string, prefixLen int) (string, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", errors.New("invalid IP address")
	}
	if prefixLen < 0 || prefixLen > 32 {
		return "", fmt.Errorf("invalid subnet mask: %d", prefixLen)
	}
	cidr := fmt.Sprintf("%s/%d", parsedIP.String(), prefixLen)
	return cidr, nil
}

func GenerateSubnet(vpcCIDR string, exitsSubnets []string) (string, error) {
	_, vpcNet, err := net.ParseCIDR(vpcCIDR)
	if err != nil {
		return "", errors.Wrapf(err, "invalid VPC CIDR")
	}

	// Get VPC mask size
	vpcMaskSize, _ := vpcNet.Mask.Size()

	// We'll use /24 as the subnet mask size (common for most use cases)
	newMaskSize := 24
	if vpcMaskSize > newMaskSize {
		return "", fmt.Errorf("VPC CIDR is smaller than target subnet size")
	}

	additionalBits := newMaskSize - vpcMaskSize
	maxSubnets := 1 << uint(additionalBits)

	// Try each possible subnet position
	for i := 0; i < maxSubnets; i++ {
		subnet, err := subnet(vpcNet, additionalBits, i)
		if err != nil {
			continue
		}

		// Check if this subnet overlaps with any existing subnets
		hasOverlap := false
		for _, existingSubnet := range exitsSubnets {
			overlap, err := IsSubnetOverlap(subnet.String(), existingSubnet)
			if err != nil || overlap {
				hasOverlap = true
				break
			}
		}

		if !hasOverlap {
			return subnet.String(), nil
		}
	}

	return "", fmt.Errorf("no available subnet found in VPC CIDR range")
}

func IsSubnetOverlap(cidr1, cidr2 string) (bool, error) {
	_, network1, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false, err
	}

	_, network2, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false, err
	}

	if network1.Contains(network2.IP) || network2.Contains(network1.IP) {
		return true, nil
	}

	return false, nil
}

func GenerateSubnets(vpcCIDR string, subnetCount int) ([]string, error) {
	_, vpcNet, err := net.ParseCIDR(vpcCIDR)
	if err != nil {
		return nil, err
	}

	subnetMaskSize, totalMaskSize := vpcNet.Mask.Size()
	requiredBits := int(math.Ceil(math.Log2(float64(subnetCount))))
	newPrefix := subnetMaskSize + requiredBits

	fmt.Println(totalMaskSize, newPrefix)

	if newPrefix > totalMaskSize {
		return nil, fmt.Errorf("subnet count too large for the given VPC CIDR. Maximum subnets that can be generated: %d", 1<<(totalMaskSize-subnetMaskSize))
	}

	subnets := []string{}
	for i := 0; i < subnetCount; i++ {
		subnet, err := subnet(vpcNet, requiredBits, i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate subnet: %v", err)
		}
		subnets = append(subnets, subnet.String())
	}

	return subnets, nil
}

func DecodeCidr(ipCidr string) string {
	_, ipnet, err := net.ParseCIDR(ipCidr)
	if err != nil {
		return ""
	}
	ip := ipnet.IP.To4()
	if ip != nil {
		return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
	}
	return ""
}

// subnet takes a parent CIDR range and creates a subnet within it
// with the given number of additional prefix bits and the given
// network number.
//
// For example, 10.3.0.0/16, extended by 8 bits, with a network number
// of 5, becomes 10.3.5.0/24 .
func subnet(base *net.IPNet, newBits int, num int) (*net.IPNet, error) {
	return subnetBig(base, newBits, big.NewInt(int64(num)))
}

// subnetBig takes a parent CIDR range and creates a subnet within it with the
// given number of additional prefix bits and the given network number. It
// differs from Subnet in that it takes a *big.Int for the num, instead of an int.
//
// For example, 10.3.0.0/16, extended by 8 bits, with a network number of 5,
// becomes 10.3.5.0/24 .
func subnetBig(base *net.IPNet, newBits int, num *big.Int) (*net.IPNet, error) {
	ip := base.IP
	mask := base.Mask

	parentLen, addrLen := mask.Size()
	newPrefixLen := parentLen + newBits

	if newPrefixLen > addrLen {
		return nil, fmt.Errorf("insufficient address space to extend prefix of %d by %d", parentLen, newBits)
	}

	maxNetNum := uint64(1<<uint64(newBits)) - 1
	if num.Uint64() > maxNetNum {
		return nil, fmt.Errorf("prefix extension of %d does not accommodate a subnet numbered %d", newBits, num)
	}

	return &net.IPNet{
		IP:   insertNumIntoIP(ip, num, newPrefixLen),
		Mask: net.CIDRMask(newPrefixLen, addrLen),
	}, nil
}

func ipToInt(ip net.IP) (*big.Int, int) {
	val := &big.Int{}
	val.SetBytes([]byte(ip))
	if len(ip) == net.IPv4len {
		return val, 32
	} else if len(ip) == net.IPv6len {
		return val, 128
	} else {
		return nil, 0
	}
}

func intToIP(ipInt *big.Int, bits int) net.IP {
	ipBytes := ipInt.Bytes()
	ret := make([]byte, bits/8)
	// Pack our IP bytes into the end of the return array,
	// since big.Int.Bytes() removes front zero padding.
	for i := 1; i <= len(ipBytes); i++ {
		ret[len(ret)-i] = ipBytes[len(ipBytes)-i]
	}
	return net.IP(ret)
}

func insertNumIntoIP(ip net.IP, bigNum *big.Int, prefixLen int) net.IP {
	ipInt, totalBits := ipToInt(ip)
	bigNum.Lsh(bigNum, uint(totalBits-prefixLen))
	ipInt.Or(ipInt, bigNum)
	return intToIP(ipInt, totalBits)
}
