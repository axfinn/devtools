package trip

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// SeedRealTrip 把"国庆 2026 粤港澳潮汕 14 日深度游"作为 fixture 落到仓储。
// 可幂等调用:已存在同名 trip 时返回 (id, false, nil),不再创建。
// 删重建可通过 --force:true 或 query 参数 force=true 触发。
//
// 数据完全来源 issue DEVT-20 描述,不做文学加工,只把散文重组为
// DayPlan + Activity 结构。
func (s *Service) SeedRealTrip(ctx context.Context, force bool) (string, bool, error) {
	const tripName = "国庆 2026 粤港澳潮汕 14 日深度游"

	// 同名检查
	existing, err := s.repo.ListTrips(ctx)
	if err != nil {
		return "", false, err
	}
	for _, t := range existing {
		if t.Name == tripName {
			if !force {
				return t.ID, false, nil
			}
			// force:删重建
			if err := s.repo.DeleteTrip(ctx, t.ID); err != nil {
				return "", false, fmt.Errorf("seed: delete existing: %w", err)
			}
			break
		}
	}

	trip := &Trip{
		Name:        tripName,
		Description: "国庆 14 天深度游,覆盖深圳/香港/澳门/珠海/广州/潮汕(汕头/潮州/揭阳)。",
		StartDate:   "2026-09-24",
		EndDate:     "2026-10-07",
		Tags:        []string{"国庆", "粤港澳", "潮汕", "家庭"},
		CoverCities: []string{"深圳", "香港", "澳门", "珠海", "广州", "潮汕"},
		NotifyEmail: "", // 用户可后续在 TripTool UI 里填
	}
	if err := s.repo.CreateTrip(ctx, trip); err != nil {
		return "", false, err
	}

	type seedActivity struct {
		Kind       ActivityKind
		Title      string
		Location   string
		Note       string
		StartTime  string
		City       string
		Region     string
		Country    string
	}

	type seedDay struct {
		Date       string
		City       string
		Region     string
		Country    string
		Activities []seedActivity
	}

	days := []seedDay{
		{
			Date: "2026-09-24", City: "深圳", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "抵达深圳宝安机场 / 深圳北站", Location: "宝安机场 / 深圳北站", Note: "入住福田 / 罗湖酒店", StartTime: "上午", City: "深圳"},
				{Kind: ActivitySight, Title: "深圳湾公园骑行 + 人才公园灯光秀", Location: "深圳湾公园", StartTime: "下午", City: "深圳"},
				{Kind: ActivitySight, Title: "平安金融中心 116 层 Free Sky 看夜景", Location: "平安金融中心", StartTime: "晚上", City: "深圳", Note: "福田 CBD 购物"},
				{Kind: ActivityFood, Title: "潮汕牛肉火锅", Location: "八合里 / 陈鹏鹏", StartTime: "晚餐", City: "深圳"},
			},
		},
		{
			Date: "2026-09-25", City: "香港", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "福田 / 罗湖口岸过关,搭东铁线到红磡", Location: "福田 / 罗湖口岸", StartTime: "上午", City: "深圳", Note: "出发前往香港"},
				{Kind: ActivitySight, Title: "尖沙咀 + 维多利亚港星光大道", Location: "尖沙咀", StartTime: "下午", City: "香港", Region: "尖沙咀"},
				{Kind: ActivitySight, Title: "天星小轮", Location: "维多利亚港", StartTime: "下午", City: "香港"},
				{Kind: ActivitySight, Title: "太平山顶 + 幻彩咏香江", Location: "太平山顶", StartTime: "晚上", City: "香港"},
				{Kind: ActivityLodging, Title: "入住尖沙咀 / 中环 / 铜锣湾", Note: "推荐住宿区", City: "香港"},
				{Kind: ActivityFood, Title: "港式茶餐厅 + 添好运点心 + 沾仔记云吞面", StartTime: "正餐", City: "香港"},
			},
		},
		{
			Date: "2026-09-26", City: "香港", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivitySight, Title: "迪士尼乐园(或海洋公园)", Location: "迪士尼乐园 / 海洋公园", Note: "选项 A:迪士尼(亲子/情侣);选项 B:海洋公园(家庭/刺激项目)", StartTime: "全天", City: "香港"},
				{Kind: ActivityShopping, Title: "铜锣湾购物 + 时代广场", Location: "铜锣湾", StartTime: "晚上", City: "香港"},
				{Kind: ActivityFood, Title: "翠华餐厅 + 一兰拉面 + 再兴烧腊", StartTime: "正餐", City: "香港"},
			},
		},
		{
			Date: "2026-09-27", City: "香港", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivitySight, Title: "中环半山扶梯 + 荷李活道 + PMQ 元创方", Location: "中环", StartTime: "上午", City: "香港", Region: "中环"},
				{Kind: ActivityFood, Title: "兰芳园 / 陆羽茶室", StartTime: "中午", City: "香港"},
				{Kind: ActivitySight, Title: "文武庙 + 庙街夜市", Location: "庙街", Note: "晚上更热闹", StartTime: "下午/晚上", City: "香港", Region: "油麻地"},
				{Kind: ActivityLeisure, Title: "旺角 / 油麻地", StartTime: "晚上", City: "香港", Region: "旺角"},
			},
		},
		{
			Date: "2026-09-28", City: "澳门", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "港澳码头 / 港珠澳大桥口岸 → 澳门", Note: "上午出发", StartTime: "上午", City: "香港"},
				{Kind: ActivitySight, Title: "大三巴 + 议事亭前地 + 玫瑰圣母堂", Location: "澳门半岛", StartTime: "下午", City: "澳门"},
				{Kind: ActivitySight, Title: "澳门塔 / 威尼斯人 / 巴黎人", StartTime: "晚上", City: "澳门"},
				{Kind: ActivityLodging, Title: "入住氹仔(巴黎人/威尼斯人/银河)", Location: "氹仔", City: "澳门", Region: "氹仔"},
				{Kind: ActivityFood, Title: "猪扒包(大利来)+ 葡式蛋挞(安德鲁/玛嘉烈)+ 葡国菜", StartTime: "正餐", City: "澳门"},
			},
		},
		{
			Date: "2026-09-29", City: "澳门", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivitySight, Title: "妈阁庙 + 港务局大楼", Location: "澳门半岛", StartTime: "上午", City: "澳门"},
				{Kind: ActivitySight, Title: "龙环葡韵 + 路氹金光大道(威尼斯人/巴黎人/伦敦人)", Location: "氹仔", StartTime: "下午", City: "澳门", Region: "氹仔"},
				{Kind: ActivitySight, Title: "永利皇宫缆车 / 表演湖", Location: "永利皇宫", StartTime: "晚上", City: "澳门"},
			},
		},
		{
			Date: "2026-09-30", City: "珠海", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "拱北口岸过关 → 珠海", StartTime: "上午", City: "澳门"},
				{Kind: ActivitySight, Title: "情侣路 + 渔女雕像 + 圆明新园", Location: "珠海", StartTime: "下午", City: "珠海"},
				{Kind: ActivityShopping, Title: "拱北口岸附近购物", StartTime: "晚上", City: "珠海"},
				{Kind: ActivityFood, Title: "横琴蚝 + 湾仔海鲜", StartTime: "正餐", City: "珠海"},
			},
		},
		{
			Date: "2026-10-01", City: "广州", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "珠海 → 广州", Note: "国庆当天抵达", StartTime: "上午"},
				{Kind: ActivitySight, Title: "广州城 / 陈家祠", Location: "陈家祠", StartTime: "上午", City: "广州"},
				{Kind: ActivitySight, Title: "沙面岛 + 上下九步行街", StartTime: "下午", City: "广州"},
				{Kind: ActivitySight, Title: "珠江夜游", Location: "珠江", StartTime: "晚上", City: "广州"},
				{Kind: ActivityLodging, Title: "入住天河 / 珠江新城", Location: "天河", City: "广州", Region: "天河"},
				{Kind: ActivityFood, Title: "早茶(点都德 / 陶陶居 / 广州酒家)", StartTime: "正餐", City: "广州"},
			},
		},
		{
			Date: "2026-10-02", City: "广州", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivitySight, Title: "白云山", Location: "白云山", StartTime: "上午", City: "广州"},
				{Kind: ActivitySight, Title: "越秀公园 + 五羊雕像 + 南越王博物院", Location: "越秀公园", StartTime: "下午", City: "广州"},
				{Kind: ActivitySight, Title: "北京路步行街 / 海珠广场", StartTime: "晚上", City: "广州"},
			},
		},
		{
			Date: "2026-10-03", City: "汕头", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "广州 → 汕头(高铁/动车)", StartTime: "上午"},
				{Kind: ActivitySight, Title: "老妈宫 + 汕头老街 + 小公园骑楼", Location: "汕头老城", StartTime: "下午", City: "汕头", Region: "老城"},
				{Kind: ActivitySight, Title: "海滨长廊", Location: "汕头海滨长廊", StartTime: "晚上", City: "汕头"},
				{Kind: ActivityFood, Title: "牛肉火锅(杏花吴记 / 海记)+ 牛肉丸 + 粿品", StartTime: "正餐", City: "汕头"},
				{Kind: ActivityLodging, Title: "入住汕头市区", City: "汕头"},
			},
		},
		{
			Date: "2026-10-04", City: "南澳岛", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "汕头 → 南澳岛", StartTime: "上午", City: "汕头"},
				{Kind: ActivitySight, Title: "南澳大桥 + 青澳湾 + 北回归线广场", Location: "青澳湾", StartTime: "上午", City: "汕头", Region: "南澳岛"},
				{Kind: ActivitySight, Title: "黄花山森林公园 + 风电场", Location: "黄花山", StartTime: "下午", City: "汕头", Region: "南澳岛"},
				{Kind: ActivitySight, Title: "青澳湾看日落 + 吃海鲜", Location: "青澳湾", StartTime: "晚上", City: "汕头", Region: "南澳岛"},
			},
		},
		{
			Date: "2026-10-05", City: "潮州", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "南澳 → 潮州", StartTime: "上午"},
				{Kind: ActivitySight, Title: "潮州古城 + 广济桥 + 韩文公祠", Location: "潮州古城", StartTime: "上午", City: "潮州"},
				{Kind: ActivityFood, Title: "潮州菜(官塘兄弟 / 潮膳楼)", StartTime: "中午", City: "潮州"},
				{Kind: ActivitySight, Title: "开元寺 + 牌坊街", Location: "牌坊街", StartTime: "下午", City: "潮州"},
				{Kind: ActivitySight, Title: "牌坊街夜景", Location: "牌坊街", StartTime: "晚上", City: "潮州"},
			},
		},
		{
			Date: "2026-10-06", City: "揭阳", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityTransit, Title: "潮州 → 揭阳", StartTime: "上午"},
				{Kind: ActivitySight, Title: "揭阳学宫 / 黄满寨瀑布", Location: "揭阳", StartTime: "上午", City: "揭阳"},
				{Kind: ActivityLeisure, Title: "返程交通缓冲 + 自由活动 / 收拾行李", Note: "下午根据返程交通调整", StartTime: "下午"},
			},
		},
		{
			Date: "2026-10-07", City: "深圳", Country: "中国",
			Activities: []seedActivity{
				{Kind: ActivityShopping, Title: "购买伴手礼", Note: "潮汕牛肉丸 / 潮州柑 / 澳门钜记饼家 / 香港美心月饼", StartTime: "上午"},
				{Kind: ActivityTransit, Title: "各自返程", Note: "下午 / 晚上根据机票 / 高铁时间", StartTime: "下午"},
			},
		},
	}

	for i, sd := range days {
		dp := &DayPlan{
			TripID: trip.ID,
			Date:   sd.Date,
			Order:  i + 1,
			Destination: Destination{
				City:    sd.City,
				Country: sd.Country,
				Region:  sd.Region,
			},
		}
		if err := s.repo.AddDay(ctx, dp); err != nil {
			return "", false, fmt.Errorf("seed: add day %s: %w", sd.Date, err)
		}
		for j, sa := range sd.Activities {
			a := &Activity{
				DayID:     dp.ID,
				Kind:      sa.Kind,
				Title:     sa.Title,
				Location:  sa.Location,
				Note:      sa.Note,
				StartTime: sa.StartTime,
				Order:     j + 1,
				Destination: Destination{
					City:    firstNonEmpty(sa.City, sd.City),
					Region:  firstNonEmpty(sa.Region, sd.Region),
					Country: firstNonEmpty(sa.Country, sd.Country),
				},
			}
			if err := s.repo.AddActivity(ctx, a); err != nil {
				return "", false, fmt.Errorf("seed: add activity %s: %w", sa.Title, err)
			}
		}
	}

	return trip.ID, true, nil
}

// ListSeededTripNames 用于 CLI / HTTP 的 seed 端点做"已存在"提示。
func (s *Service) ListSeededTripNames(ctx context.Context) ([]string, error) {
	trips, err := s.repo.ListTrips(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(trips))
	for _, t := range trips {
		out = append(out, t.Name)
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// 防止 import errors 漂移:在某些精简 build tag 下 errors 包可能未被使用,
// 这里放一个占位符 import 让 go vet 不抱怨 unused。
var _ = errors.New