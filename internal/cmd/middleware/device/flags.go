package device

import (
	"jpy-cli/pkg/middleware/device/selector"

	"github.com/spf13/cobra"
)

// CommonFlags 定义了所有设备相关命令的统一筛选参数
type CommonFlags struct {
	Group         string
	ServerPattern string
	UUID          string
	IP            string
	Seat          int

	// 筛选器
	FilterADB    string // "true"/"false"
	FilterUSB    string // "true"/"false"
	FilterOnline string // "true"/"false"
	FilterHasIP  string // "true"/"false"
	FilterUUID   string // "true"/"false"

	AuthorizedOnly string // "true"/"false"/""

	Interactive bool
	All         bool   // 跳过交互模式并处理所有匹配设备
	Output      string // 输出模式: tui/plain/json
}

// AddCommonFlags 为命令添加统一的筛选参数
func AddCommonFlags(cmd *cobra.Command, opts *CommonFlags) {
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "目标服务器分组")
	cmd.Flags().StringVarP(&opts.ServerPattern, "server", "s", "", "服务器地址匹配模式 (例如: 192.168.1)")
	cmd.Flags().StringVarP(&opts.UUID, "uuid", "u", "", "设备UUID (模糊匹配)")
	cmd.Flags().StringVar(&opts.IP, "ip", "", "设备IP (模糊匹配)")
	cmd.Flags().IntVar(&opts.Seat, "seat", -1, "机位号")

	cmd.Flags().StringVar(&opts.FilterADB, "filter-adb", "", "筛选ADB状态 (true/false)")
	cmd.Flags().StringVar(&opts.FilterUSB, "filter-usb", "", "筛选USB模式 (true=USB, false=OTG)")
	cmd.Flags().StringVar(&opts.FilterOnline, "filter-online", "", "筛选在线状态 (true/false)")
	cmd.Flags().StringVar(&opts.FilterHasIP, "filter-has-ip", "", "筛选IP存在状态 (true/false)")
	cmd.Flags().StringVar(&opts.FilterUUID, "filter-uuid", "", "筛选UUID存在状态 (true/false)")

	cmd.Flags().StringVar(&opts.AuthorizedOnly, "authorized", "", "筛选授权状态 (true/false)")
	// 允许 --authorized 等同于 --authorized=true
	cmd.Flags().Lookup("authorized").NoOptDefVal = "true"

	cmd.Flags().BoolVarP(&opts.Interactive, "interactive", "i", false, "交互式选择模式")
	cmd.Flags().BoolVar(&opts.All, "all", false, "跳过交互模式并处理所有匹配设备")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "tui", "输出模式 (tui/plain/json)")
}

// ToSelectorOptions 将通用Flag转换为Selector选项
func (opts *CommonFlags) ToSelectorOptions() (selector.SelectionOptions, error) {
	res := selector.SelectionOptions{
		Group:         opts.Group,
		ServerPattern: opts.ServerPattern,
		UUID:          opts.UUID,
		IP:            opts.IP,
		Seat:          opts.Seat,
		Interactive:   opts.Interactive,
	}

	if opts.AuthorizedOnly != "" {
		val := opts.AuthorizedOnly == "true"
		res.AuthorizedOnly = &val
	}

	if opts.FilterADB != "" {
		val := opts.FilterADB == "true"
		res.ADB = &val
	}
	if opts.FilterUSB != "" {
		val := opts.FilterUSB == "true"
		res.USB = &val
	}
	if opts.FilterOnline != "" {
		val := opts.FilterOnline == "true"
		res.BizOnline = &val // 映射到业务在线
	}
	if opts.FilterHasIP != "" {
		val := opts.FilterHasIP == "true"
		res.HasIP = &val
	}
	if opts.FilterUUID != "" {
		val := opts.FilterUUID == "true"
		res.HasUUID = &val
	}
	return res, nil
}
