package utils

import (
	"fmt"
	"math"
	"math/big"
	"net"

	"github.com/pkg/errors"
)

func GenerateClusterCIDR(clusterID int64) (string, error) {
	if clusterID <= 0 {
		return "", errors.New("cluster ID must be positive")
	}
	if clusterID > 255 {
		return "", errors.New("cluster ID exceeds maximum allowed value (255)")
	}
	cidr := fmt.Sprintf("10.%d.0.0/16", clusterID)
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
	for i := range make([]struct{}, maxSubnets) {
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

func GenerateSubnets(vpcCIDR string, subnetCount int) ([]string, error) {
	_, vpcNet, err := net.ParseCIDR(vpcCIDR)
	if err != nil {
		return nil, err
	}

	subnetMaskSize, totalMaskSize := vpcNet.Mask.Size()
	requiredBits := int(math.Ceil(math.Log2(float64(subnetCount))))
	newPrefix := subnetMaskSize + requiredBits

	if newPrefix > totalMaskSize {
		return nil, fmt.Errorf("subnet count too large for the given VPC CIDR. Maximum subnets that can be generated: %d", 1<<(totalMaskSize-subnetMaskSize))
	}

	subnets := []string{}
	for i := range make([]struct{}, subnetCount) {
		subnet, err := subnet(vpcNet, requiredBits, i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate subnet: %v", err)
		}
		subnets = append(subnets, subnet.String())
	}

	return subnets, nil
}

type KubernetesCIDRs struct {
	PodCIDR     string
	ServiceCIDR string
}

func GenerateKubernetesCIDRs(clusterID int64, vpcCIDR string) (*KubernetesCIDRs, error) {
	if clusterID <= 0 {
		return nil, errors.New("cluster ID must be positive")
	}

	if clusterID > 255 {
		return nil, errors.New("cluster ID exceeds maximum allowed value (255)")
	}

	_, _, err := net.ParseCIDR(vpcCIDR)
	if err != nil {
		return nil, errors.Wrap(err, "invalid VPC CIDR")
	}

	podCandidates := []string{
		fmt.Sprintf("172.%d.0.0/16", clusterID),
		fmt.Sprintf("192.168.%d.0/24", clusterID),
	}
	serviceCandidates := []string{
		fmt.Sprintf("10.%d.0.0/16", 96+clusterID),
		fmt.Sprintf("10.%d.0.0/16", 160+clusterID),
	}

	var podCIDR string
	for _, candidate := range podCandidates {
		overlap, err := IsSubnetOverlap(candidate, vpcCIDR)
		if err != nil {
			continue
		}
		if !overlap {
			podCIDR = candidate
			break
		}
	}
	if podCIDR == "" {
		return nil, errors.New("unable to find non-overlapping Pod CIDR")
	}

	var serviceCIDR string
	for _, candidate := range serviceCandidates {
		overlapVPC, err1 := IsSubnetOverlap(candidate, vpcCIDR)
		overlapPod, err2 := IsSubnetOverlap(candidate, podCIDR)
		if err1 != nil || err2 != nil {
			continue
		}
		if !overlapVPC && !overlapPod {
			serviceCIDR = candidate
			break
		}
	}
	if serviceCIDR == "" {
		return nil, errors.New("unable to find non-overlapping Service CIDR")
	}

	return &KubernetesCIDRs{
		PodCIDR:     podCIDR,
		ServiceCIDR: serviceCIDR,
	}, nil
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

// CalculateCIDRIPCount 计算CIDR中可用的IP地址数量
func CalculateCIDRIPCount(cidr string) (uint64, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, errors.Wrap(err, "invalid CIDR")
	}

	// 获取掩码的位数
	ones, bits := ipnet.Mask.Size()
	// 2的(位数差)次方
	return uint64(1) << uint(bits-ones), nil
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
