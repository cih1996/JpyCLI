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

输出模式:
  --output tui     写入文件（默认，需指定文件路径）
  --output plain   输出到 stdout，制表符分隔
  --output json    输出到 stdout，JSON 格式

支持导出的字段：
- --export-id: 设备ID (根据服务器地址生成)
- --export-ip: 设备IP地址
- --export-uuid: 设备序列号
- --export-seat: 设备机位号

如果未指定任何导出字段，默认导出所有字段。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				outputFile = args[0]
			}

			// 非 stdout 输出模式需要文件路径
			if opts.Output == "tui" && outputFile == "" {
				return fmt.Errorf("请指定输出文件路径，或使用 -o plain/json 输出到标准输出")
			}

			if !exportID && !exportIP && !exportUUID && !exportSeat && !exportAuto {
				exportID = true
				exportUUID = true
				exportIP = true
				exportSeat = true
			}

			selOpts, err := opts.ToSelectorOptions()
			if err != nil {
				return err
			}
			selOpts.Interactive = false
			// 非 TUI 模式下跳过 BubbleTea 进度条，防止 Win7 SSH 崩溃
			if opts.Output == "plain" || opts.Output == "json" {
				selOpts.Silent = true
			}

			devices, err := selector.SelectDevices(selOpts)
			if err != nil {
				return err
			}

			if len(devices) == 0 {
				if opts.Output == "json" {
					fmt.Println("{\"total\":0,\"devices\":[]}")
					return nil
				}
				return fmt.Errorf("没有找到符合条件的设备")
			}

			sort.Slice(devices, func(i, j int) bool {
				if devices[i].ServerIndex != devices[j].ServerIndex {
					return devices[i].ServerIndex < devices[j].ServerIndex
				}
				return devices[i].Seat < devices[j].Seat
			})

			// stdout 输出模式
			switch opts.Output {
			case "json":
				printExportJSON(devices)
				return nil
			case "plain":
				printExportPlain(devices)
				return nil
			}

			// 文件输出模式（tui 默认）
			file, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("创建文件失败: %v", err)
			}
			defer file.Close()

			var stats = struct {
				totalDevices    int
				exportedDevices int
				missingUUID     int
				missingIP       int
				fixedIP         int
			}{}

			stats.totalDevices = len(devices)

			for _, d := range devices {
				if exportAuto {
					if d.UUID != "" && d.IP == "" {
						fixedIP := autoCompleteIP(devices, d)
						if fixedIP != "" {
							d.IP = fixedIP
							stats.fixedIP++
							var fields []string
							fields = append(fields, generateDeviceID(d.ServerURL))
							fields = append(fields, d.UUID)
							fields = append(fields, d.IP)
							fields = append(fields, strconv.Itoa(d.Seat))

							line := strings.Join(fields, "\t") + "\n"
							if _, err := file.WriteString(line); err != nil {
								return fmt.Errorf("写入文件失败: %v", err)
							}
							stats.exportedDevices++
						} else {
							stats.missingIP++
						}
					} else {
						if d.UUID == "" {
							stats.missingUUID++
						}
						continue
					}
				} else {
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
	url := strings.TrimPrefix(serverURL, "https://")
	url = strings.TrimPrefix(url, "http://")

	hostPort := strings.Split(url, ":")
	var host, port string

	if len(hostPort) == 1 {
		host = hostPort[0]
		port = "443"
	} else if len(hostPort) == 2 {
		host = hostPort[0]
		port = hostPort[1]
	} else {
		return "unknown"
	}

	if port == "443" {
		ipParts := strings.Split(host, ".")
		if len(ipParts) >= 4 {
			return ipParts[2] + ipParts[3]
		}
		return host
	}

	return port
}

// autoCompleteIP 智能补齐缺失的IP地址
func autoCompleteIP(devices []model.DeviceInfo, currentDevice model.DeviceInfo) string {
	groupedDevices := make(map[string][]model.DeviceInfo)
	for _, d := range devices {
		if d.UUID != "" && d.IP != "" {
			groupedDevices[d.ServerURL] = append(groupedDevices[d.ServerURL], d)
		}
	}

	for serverURL := range groupedDevices {
		sort.Slice(groupedDevices[serverURL], func(i, j int) bool {
			return groupedDevices[serverURL][i].Seat < groupedDevices[serverURL][j].Seat
		})
	}

	serverDevices, exists := groupedDevices[currentDevice.ServerURL]
	if !exists || len(serverDevices) == 0 {
		return ""
	}

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

	if foundPrev && foundNext {
		if isSameNetwork(prevDevice.IP, nextDevice.IP) {
			return interpolateIP(prevDevice.IP, nextDevice.IP, prevDevice.Seat, nextDevice.Seat, currentDevice.Seat)
		}
	}

	if foundPrev {
		return incrementIP(prevDevice.IP, currentDevice.Seat-prevDevice.Seat)
	}

	if foundNext {
		return decrementIP(nextDevice.IP, nextDevice.Seat-currentDevice.Seat)
	}

	return ""
}

func isSameNetwork(ip1, ip2 string) bool {
	parts1 := strings.Split(ip1, ".")
	parts2 := strings.Split(ip2, ".")
	if len(parts1) != 4 || len(parts2) != 4 {
		return false
	}
	return parts1[0] == parts2[0] && parts1[1] == parts2[1] && parts1[2] == parts2[2]
}

func interpolateIP(ip1, ip2 string, seat1, seat2, targetSeat int) string {
	parts1 := strings.Split(ip1, ".")
	parts2 := strings.Split(ip2, ".")
	if len(parts1) != 4 || len(parts2) != 4 {
		return ""
	}

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
	ipPerSeat := float64(ipDiff) / float64(seatDiff)
	targetIPChange := int(ipPerSeat * float64(targetSeat-seat1))

	targetLastOctet := lastOctet1 + targetIPChange
	if targetLastOctet < 0 || targetLastOctet > 255 {
		return ""
	}

	return fmt.Sprintf("%s.%s.%s.%d", parts1[0], parts1[1], parts1[2], targetLastOctet)
}

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
