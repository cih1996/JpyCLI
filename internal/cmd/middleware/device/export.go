package device

import (
	"fmt"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/model"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NewExportCmd() *cobra.Command {
	var (
		exportID   bool
		exportIP   bool
		exportUUID bool
		exportSeat bool
		exportAuto bool
		outputFile string
	)
	opts := CommonFlags{}

	cmd := &cobra.Command{
		Use:   "export [output-file]",
		Short: "导出设备信息到文件",
		Long: `导出设备信息到指定文件，支持选择导出的字段。

支持导出的字段：
- --id: 设备ID (根据服务器地址生成)
- --ip: 设备IP地址
- --uuid: 设备序列号
- --seat: 设备机位号

如果未指定任何导出字段，默认导出所有字段，格式为：ID\tUUID\tIP\tSeat

ID生成规则：
- 局域网服务器 (端口443): 使用IP地址的后3、4段，如192.168.12.201:443 -> 12201
- 穿透服务器: 使用穿透端口作为ID，如129.204.22.176:12201 -> 12201`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 获取输出文件路径
			if len(args) > 0 {
				outputFile = args[0]
			}
			if outputFile == "" {
				return fmt.Errorf("请指定输出文件路径")
			}

			// 如果没有指定任何导出字段，默认导出所有
			if !exportID && !exportIP && !exportUUID && !exportSeat && !exportAuto {
				exportID = true
				exportUUID = true
				exportIP = true
				exportSeat = true
			}

			// 获取设备列表
			selOpts, err := opts.ToSelectorOptions()
			if err != nil {
				return err
			}
			selOpts.Interactive = false

			devices, err := selector.SelectDevices(selOpts)
			if err != nil {
				return err
			}

			if len(devices) == 0 {
				return fmt.Errorf("没有找到符合条件的设备")
			}

			// 排序: 服务器顺序 -> 机位
			sort.Slice(devices, func(i, j int) bool {
				if devices[i].ServerIndex != devices[j].ServerIndex {
					return devices[i].ServerIndex < devices[j].ServerIndex
				}
				return devices[i].Seat < devices[j].Seat
			})

			// 创建输出文件
			file, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("创建文件失败: %v", err)
			}
			defer file.Close()

			// 写入数据
			var stats = struct {
				totalDevices    int
				exportedDevices int
				missingUUID     int
				missingIP       int
				fixedIP         int
			}{}

			stats.totalDevices = len(devices)

			for _, d := range devices {
				// 智能导出模式逻辑
				if exportAuto {
					// 只处理有UUID但缺失IP的设备
					if d.UUID != "" && d.IP == "" {
						// 智能补齐IP
						fixedIP := autoCompleteIP(devices, d)
						if fixedIP != "" {
							d.IP = fixedIP
							stats.fixedIP++
							// 只导出补齐成功的设备
							var fields []string
							fields = append(fields, generateDeviceID(d.ServerURL))
							fields = append(fields, d.UUID)
							fields = append(fields, d.IP)
							fields = append(fields, strconv.Itoa(d.Seat))

							// 用制表符分隔字段
							line := strings.Join(fields, "\t") + "\n"
							if _, err := file.WriteString(line); err != nil {
								return fmt.Errorf("写入文件失败: %v", err)
							}
							stats.exportedDevices++
						} else {
							stats.missingIP++
							// 无法补齐IP，跳过导出
						}
					} else {
						// 其他情况（有IP或有UUID但已有IP，或没有UUID）都跳过导出
						if d.UUID == "" {
							stats.missingUUID++
						}
						continue
					}
				} else {
					// 非智能导出模式，保持原有逻辑
					var fields []string

					if exportID {
						fields = append(fields, generateDeviceID(d.ServerURL))
					}
					if exportUUID {
						fields = append(fields, d.UUID)
					}
					if exportIP {
						fields = append(fields, d.IP)
					}
					if exportSeat {
						fields = append(fields, strconv.Itoa(d.Seat))
					}

					// 用制表符分隔字段
					line := strings.Join(fields, "\t") + "\n"
					if _, err := file.WriteString(line); err != nil {
						return fmt.Errorf("写入文件失败: %v", err)
					}
					stats.exportedDevices++
				}
			}

			fmt.Printf("成功导出 %d 台设备信息到: %s\n", stats.exportedDevices, outputFile)
			return nil
		},
	}

	AddCommonFlags(cmd, &opts)
	cmd.Flags().BoolVar(&exportID, "export-id", false, "导出设备ID")
	cmd.Flags().BoolVar(&exportIP, "export-ip", false, "导出设备IP地址")
	cmd.Flags().BoolVar(&exportUUID, "export-uuid", false, "导出设备序列号")
	cmd.Flags().BoolVar(&exportSeat, "export-seat", false, "导出设备机位号")
	cmd.Flags().BoolVar(&exportAuto, "export-auto", false, "智能导出模式: 自动补齐缺失的IP地址，只导出有UUID的设备")

	return cmd
}

// generateDeviceID 根据服务器URL生成设备ID
func generateDeviceID(serverURL string) string {
	// 清理URL前缀
	url := strings.TrimPrefix(serverURL, "https://")
	url = strings.TrimPrefix(url, "http://")

	// 分离主机和端口
	hostPort := strings.Split(url, ":")
	var host, port string

	if len(hostPort) == 1 {
		// 没有端口号，默认使用443
		host = hostPort[0]
		port = "443"
	} else if len(hostPort) == 2 {
		// 有端口号
		host = hostPort[0]
		port = hostPort[1]
	} else {
		return "unknown"
	}

	// 判断是否为局域网服务器 (端口443)
	if port == "443" {
		// 提取IP地址的后3、4段
		ipParts := strings.Split(host, ".")
		if len(ipParts) >= 4 {
			return ipParts[2] + ipParts[3] // 例如: 192.168.12.201 -> "12201"
		}
		return host
	}

	// 穿透服务器: 使用端口作为ID
	return port
}

// autoCompleteIP 智能补齐缺失的IP地址
func autoCompleteIP(devices []model.DeviceInfo, currentDevice model.DeviceInfo) string {
	// 按服务器分组和机位排序的设备
	groupedDevices := make(map[string][]model.DeviceInfo)
	for _, d := range devices {
		if d.UUID != "" && d.IP != "" {
			groupedDevices[d.ServerURL] = append(groupedDevices[d.ServerURL], d)
		}
	}

	// 对每个服务器的设备按机位排序
	for serverURL := range groupedDevices {
		sort.Slice(groupedDevices[serverURL], func(i, j int) bool {
			return groupedDevices[serverURL][i].Seat < groupedDevices[serverURL][j].Seat
		})
	}

	// 查找当前设备所在服务器的其他设备
	serverDevices, exists := groupedDevices[currentDevice.ServerURL]
	if !exists || len(serverDevices) == 0 {
		return "" // 没有参考设备，无法补齐
	}

	// 查找当前设备机位前后的设备
	var prevDevice, nextDevice model.DeviceInfo
	foundPrev, foundNext := false, false
	for _, d := range serverDevices {
		if d.Seat < currentDevice.Seat {
			prevDevice = d
			foundPrev = true
		} else if d.Seat > currentDevice.Seat && !foundNext {
			nextDevice = d
			foundNext = true
		}
	}

	// 尝试根据前后设备的IP推断当前设备的IP
	if foundPrev && foundNext {
		// 前后都有设备，取中间值
		prevIP := prevDevice.IP
		nextIP := nextDevice.IP

		if isSameNetwork(prevIP, nextIP) {
			return interpolateIP(prevIP, nextIP, prevDevice.Seat, nextDevice.Seat, currentDevice.Seat)
		}
	}

	if foundPrev {
		// 只有前一个设备，递增IP
		return incrementIP(prevDevice.IP, currentDevice.Seat-prevDevice.Seat)
	}

	if foundNext {
		// 只有后一个设备，递减IP
		return decrementIP(nextDevice.IP, nextDevice.Seat-currentDevice.Seat)
	}

	return "" // 无法推断
}

// isSameNetwork 检查两个IP是否在同一网络段
func isSameNetwork(ip1, ip2 string) bool {
	parts1 := strings.Split(ip1, ".")
	parts2 := strings.Split(ip2, ".")

	if len(parts1) != 4 || len(parts2) != 4 {
		return false
	}

	// 检查前三个段是否相同
	return parts1[0] == parts2[0] && parts1[1] == parts2[1] && parts1[2] == parts2[2]
}

// interpolateIP 在两个IP之间插值
func interpolateIP(ip1, ip2 string, seat1, seat2, targetSeat int) string {
	parts1 := strings.Split(ip1, ".")
	parts2 := strings.Split(ip2, ".")

	if len(parts1) != 4 || len(parts2) != 4 {
		return ""
	}

	// 计算插值比例
	seatDiff := seat2 - seat1
	if seatDiff == 0 {
		return ""
	}

	lastOctet1, err1 := strconv.Atoi(parts1[3])
	lastOctet2, err2 := strconv.Atoi(parts2[3])
	if err1 != nil || err2 != nil {
		return ""
	}

	ipDiff := lastOctet2 - lastOctet1

	// 计算每个机位对应的IP变化量
	ipPerSeat := float64(ipDiff) / float64(seatDiff)
	targetIPChange := int(ipPerSeat * float64(targetSeat-seat1))

	targetLastOctet := lastOctet1 + targetIPChange
	if targetLastOctet < 0 || targetLastOctet > 255 {
		return ""
	}

	return fmt.Sprintf("%s.%s.%s.%d", parts1[0], parts1[1], parts1[2], targetLastOctet)
}

// incrementIP 递增IP地址
func incrementIP(baseIP string, increment int) string {
	parts := strings.Split(baseIP, ".")
	if len(parts) != 4 {
		return ""
	}

	lastOctet, err := strconv.Atoi(parts[3])
	if err != nil {
		return ""
	}

	newLastOctet := lastOctet + increment
	if newLastOctet < 0 || newLastOctet > 255 {
		return ""
	}

	return fmt.Sprintf("%s.%s.%s.%d", parts[0], parts[1], parts[2], newLastOctet)
}

// decrementIP 递减IP地址
func decrementIP(baseIP string, decrement int) string {
	parts := strings.Split(baseIP, ".")
	if len(parts) != 4 {
		return ""
	}

	lastOctet, err := strconv.Atoi(parts[3])
	if err != nil {
		return ""
	}

	newLastOctet := lastOctet - decrement
	if newLastOctet < 0 || newLastOctet > 255 {
		return ""
	}

	return fmt.Sprintf("%s.%s.%s.%d", parts[0], parts[1], parts[2], newLastOctet)
}
