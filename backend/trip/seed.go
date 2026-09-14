package trip

import (
	"context"
	"errors"
	"fmt"
)

// SeedFirstTrip 把"国庆 2026 粤港澳潮汕 14 日深度游"这条真实行程
// 落到仓储里(若已存在同名则跳过 —— 用 Name 唯一性检测)。
//
// 14 天行程覆盖:深圳 → 香港 → 澳门 → 珠海 → 广州 → 汕头(南澳) → 潮州 → 揭阳 散团。
// 数据来自用户提供的攻略原文,按 DayPlan + Activity 结构拆分落地。
//
// 调用时机:CLI 启动时自动 seed;`trip seed` 子命令可显式触发。
func SeedFirstTrip(ctx context.Context, svc *Service) (*Trip, bool, error) {
	const seedName = "国庆 2026 粤港澳潮汕 14 日深度游"

	existing, err := svc.List(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("seed: list trips: %w", err)
	}
	for _, t := range existing {
		if t.Name == seedName {
			loaded, err := svc.Show(ctx, t.ID)
			if err != nil {
				return nil, false, err
			}
			return loaded, false, nil
		}
	}

	t, err := svc.CreateTrip(ctx, CreateTripInput{
		Name:      seedName,
		StartDate: "2026-09-24",
		EndDate:   "2026-10-07",
		Summary:   "14 天深度游,串联深圳集合 → 香港 → 澳门 → 珠海 → 广州 → 汕头/南澳 → 潮州 → 揭阳散团。亲子/情侣/家庭通用,覆盖口岸过关、迪士尼/海洋公园选项、葡式蛋挞、早茶、牛肉火锅、海岛日落、潮州古城等高光节点。",
		Cities:    "深圳,香港,澳门,珠海,广州,汕头,南澳岛,潮州,揭阳",
		Tags:      "国庆,亲子,美食,海岛,跨境,深度游",
	})
	if err != nil {
		return nil, false, fmt.Errorf("seed: create trip: %w", err)
	}

	for _, day := range seedDays {
		d, err := svc.AddDay(ctx, AddDayInput{
			TripID:  t.ID,
			Date:    day.date,
			City:    day.city,
			Title:   day.title,
			Summary: day.summary,
		})
		if err != nil {
			return nil, false, fmt.Errorf("seed: add day %s: %w", day.date, err)
		}
		for _, a := range day.activities {
			if _, err := svc.AddActivity(ctx, AddActivityInput{
				DayPlanID: d.ID,
				Kind:      a.kind,
				Time:      a.time,
				Title:     a.title,
				Location:  a.location,
				Notes:     a.notes,
			}); err != nil {
				return nil, false, fmt.Errorf("seed: add activity %s/%s: %w", day.date, a.title, err)
			}
		}
	}

	loaded, err := svc.Show(ctx, t.ID)
	if err != nil {
		return nil, false, err
	}
	return loaded, true, nil
}

// seedDay 内部使用的 seed 单元结构。
type seedDay struct {
	date       string
	city       string
	title      string
	summary    string
	activities []seedActivity
}

// seedActivity 单个 activity 的 seed 数据。
type seedActivity struct {
	kind     ActivityKind
	time     string
	title    string
	location string
	notes    string
}

// seedDays —— 14 天,按用户原文拆分。
var seedDays = []seedDay{
	{
		date:    "2026-09-24",
		city:    "深圳",
		title:   "D1 深圳集合日",
		summary: "落地宝安机场或深圳北站,入住福田/罗湖,晚上看 CBD 夜景。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "全天", title: "抵达深圳宝安机场 / 深圳北站", location: "宝安机场 / 深圳北站", notes: "入住福田 / 罗湖酒店,靠近口岸便于后续过关"},
			{kind: ActivitySight, time: "下午", title: "深圳湾公园骑行 + 人才公园灯光秀", location: "深圳湾公园 / 人才公园", notes: "傍晚骑行追日落,灯光秀 19:00 / 20:00 各一场"},
			{kind: ActivitySight, time: "晚上", title: "平安金融中心 116 层 Free Sky 看夜景", location: "福田 CBD 平安金融中心", notes: "建议提前在官方小程序预约门票"},
			{kind: ActivityFood, time: "晚上", title: "潮汕牛肉火锅", location: "福田 / 罗湖", notes: "推荐:八合里、陈鹏鹏 —— 现切牛肉三吊水"},
		},
	},
	{
		date:    "2026-09-25",
		city:    "香港",
		title:   "D2 深圳 → 香港",
		summary: "上午过关走东铁,下午尖沙咀 + 维港,晚上太平山顶看幻彩咏香江。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "福田 / 罗湖口岸过关", location: "福田口岸 / 罗湖口岸", notes: "建议福田口岸过 → 落马洲 → 东铁线红磡,避开罗湖早高峰"},
			{kind: ActivitySight, time: "下午", title: "尖沙咀 + 维多利亚港星光大道", location: "尖沙咀", notes: "李小龙铜像、天星小轮钟楼、1881 Heritage"},
			{kind: ActivitySight, time: "下午", title: "天星小轮过海", location: "尖沙咀 ↔ 中环 / 湾仔", notes: "百年小轮,3 港币一程,推荐上层前排"},
			{kind: ActivitySight, time: "晚上", title: "太平山顶 + 幻彩咏香江", location: "太平山顶", notes: "建议搭 15 路巴士上山,8 点前到位看灯光秀"},
			{kind: ActivityLodging, time: "晚上", title: "入住尖沙咀 / 中环 / 铜锣湾", location: "尖沙咀 / 中环 / 铜锣湾", notes: "三选一,看次日行程动线"},
			{kind: ActivityFood, time: "全天", title: "港式茶餐厅 / 添好运点心 / 沾仔记云吞面", location: "中环 / 尖沙咀", notes: "添好运推荐酥皮叉烧包、虾饺;沾仔记必点鲜虾云吞"},
		},
	},
	{
		date:    "2026-09-26",
		city:    "香港",
		title:   "D3 香港迪士尼或海洋公园",
		summary: "主题乐园二选一,晚上铜锣湾购物。",
		activities: []seedActivity{
			{kind: ActivityNote, time: "全天", title: "选项 A:迪士尼乐园 / 选项 B:海洋公园", location: "大屿山 / 香港仔", notes: "A 适合亲子 / 情侣,烟花 20:30;B 适合家庭 / 刺激项目(越矿飞车、极速之旅)"},
			{kind: ActivityShopping, time: "晚上", title: "铜锣湾购物 + 时代广场", location: "铜锣湾", notes: "崇光百货 SOGO 周年庆通常在 9-10 月,留意折扣"},
			{kind: ActivityFood, time: "全天", title: "翠华餐厅 / 一兰拉面 / 再兴烧腊", location: "尖沙咀 / 中环 / 铜锣湾", notes: "翠华菠萝油 + 奶茶是港味标配"},
		},
	},
	{
		date:    "2026-09-27",
		city:    "香港",
		title:   "D4 香港经典一日",
		summary: "中环半山扶梯 → PMQ → 文武庙 → 庙街夜市。",
		activities: []seedActivity{
			{kind: ActivitySight, time: "上午", title: "中环半山扶梯 + 荷李活道 + PMQ 元创方", location: "中环", notes: "全球最长户外扶梯系统,慢慢逛 + Soho 区涂鸦墙"},
			{kind: ActivityFood, time: "中午", title: "兰芳园 / 陆羽茶室", location: "中环", notes: "兰芳园丝袜奶茶 + 猪扒包;陆羽茶室需订位"},
			{kind: ActivitySight, time: "下午", title: "文武庙", location: "上环", notes: "百年庙宇,中央燃着巨型塔香"},
			{kind: ActivitySight, time: "晚上", title: "庙街夜市 + 旺角 / 油麻地", location: "油麻地 / 旺角", notes: "庙街 19 点后开档,大排档 + 算命 + 歌厅文化"},
		},
	},
	{
		date:    "2026-09-28",
		city:    "澳门",
		title:   "D5 香港 → 澳门",
		summary: "上午港澳码头 / 港珠澳大桥过关,下午大三巴,晚上路氹金光大道。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "港澳码头 / 港珠澳大桥口岸到澳门", location: "上环港澳码头 / 港珠澳大桥香港口岸", notes: "高铁 + 金巴也是选项;带好港澳通行证 + 签注"},
			{kind: ActivitySight, time: "下午", title: "大三巴 + 议事亭前地 + 玫瑰圣母堂", location: "澳门半岛", notes: "大三巴是圣保禄教堂遗址,旁边恋爱巷很出片"},
			{kind: ActivitySight, time: "晚上", title: "澳门塔 / 威尼斯人 / 巴黎人", location: "氹仔 / 路氹金光大道", notes: "澳门塔看日落 + 蹦极;威尼斯人贡多拉游船"},
			{kind: ActivityLodging, time: "晚上", title: "入住氹仔(巴黎人 / 威尼斯人 / 银河)", location: "氹仔", notes: "三家连成一片,免去拖行李奔波"},
			{kind: ActivityFood, time: "全天", title: "猪扒包 + 葡式蛋挞 + 葡国菜", location: "澳门半岛 / 氹仔", notes: "猪扒包推荐大利来记;葡挞安德鲁 / 玛嘉烈二选一"},
		},
	},
	{
		date:    "2026-09-29",
		city:    "澳门",
		title:   "D6 澳门深度",
		summary: "妈阁庙 + 龙环葡韵 + 路氹金光大道看表演湖。",
		activities: []seedActivity{
			{kind: ActivitySight, time: "上午", title: "妈阁庙 + 港务局大楼", location: "澳门半岛", notes: "妈阁庙是 Macau 名字来源,港务局大楼是摩尔式建筑"},
			{kind: ActivitySight, time: "下午", title: "龙环葡韵 + 路氹金光大道", location: "氹仔", notes: "龙环葡韵 5 栋薄荷绿小别墅;路氹连看威尼斯人 / 巴黎人 / 伦敦人"},
			{kind: ActivitySight, time: "晚上", title: "永利皇宫缆车 + 表演湖", location: "永利皇宫", notes: "免费缆车观景,表演湖每 30 分钟一场"},
		},
	},
	{
		date:    "2026-09-30",
		city:    "珠海",
		title:   "D7 澳门 → 珠海",
		summary: "拱北口岸过关,下午情侣路 + 圆明新园。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "拱北口岸过关到珠海", location: "拱北口岸", notes: "过关高峰 8-10 点,提早出门"},
			{kind: ActivitySight, time: "下午", title: "情侣路 + 渔女雕像 + 圆明新园", location: "香洲区", notes: "情侣路沿海步行 4 公里,渔女是珠海地标"},
			{kind: ActivityShopping, time: "晚上", title: "拱北口岸附近购物", location: "拱北商圈", notes: "口岸地下商场免税品 + 日韩药妆"},
			{kind: ActivityFood, time: "全天", title: "横琴蚝 + 湾仔海鲜", location: "横琴 / 湾仔", notes: "横琴蚝肥美清蒸最佳"},
		},
	},
	{
		date:    "2026-10-01",
		city:    "广州",
		title:   "D8 珠海 → 广州(国庆当天)",
		summary: "上午陈家祠,下午沙面 + 上下九,晚上珠江夜游。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "珠海 → 广州(高铁 / 大巴)", location: "珠海站 → 广州南站", notes: "广珠城轨 1 小时直达广州南"},
			{kind: ActivitySight, time: "上午", title: "陈家祠", location: "荔湾区", notes: "岭南建筑艺术明珠,看灰塑 / 砖雕 / 木雕"},
			{kind: ActivitySight, time: "下午", title: "沙面岛 + 上下九步行街", location: "荔湾区", notes: "沙面欧陆建筑群适合拍照;上下九吃老字号"},
			{kind: ActivitySight, time: "晚上", title: "珠江夜游", location: "天字码头 / 大沙头码头", notes: "夜游船 70-90 分钟,看小蛮腰 + 海心桥"},
			{kind: ActivityLodging, time: "晚上", title: "入住天河 / 珠江新城", location: "天河区 / 珠江新城", notes: "地铁 1 / 3 号线沿线方便次日"},
			{kind: ActivityFood, time: "全天", title: "广州早茶", location: "荔湾 / 天河", notes: "推荐:点都德、陶陶居、广州酒家 —— 虾饺 / 烧卖 / 叉烧包"},
		},
	},
	{
		date:    "2026-10-02",
		city:    "广州",
		title:   "D9 广州经典一日",
		summary: "白云山 + 越秀公园 + 南越王博物院 + 北京路。",
		activities: []seedActivity{
			{kind: ActivitySight, time: "上午", title: "白云山", location: "白云区", notes: "索道上下山,摩星岭看广州全景"},
			{kind: ActivitySight, time: "下午", title: "越秀公园 + 五羊雕像 + 南越王博物院", location: "越秀区", notes: "五羊雕像是广州城标;南越王墓出土文物必看"},
			{kind: ActivityShopping, time: "晚上", title: "北京路步行街 / 海珠广场", location: "越秀区", notes: "千年古道遗址在步行街玻璃栈道下"},
		},
	},
	{
		date:    "2026-10-03",
		city:    "汕头",
		title:   "D10 广州 → 潮汕(汕头)",
		summary: "高铁到汕头,下午老街骑楼,晚上海滨长廊。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "高铁 / 动车广州 → 汕头", location: "广州南 → 汕头站", notes: "约 3 小时,班次多"},
			{kind: ActivitySight, time: "下午", title: "老妈宫 + 汕头老街 + 小公园骑楼", location: "金平区", notes: "老妈宫是潮汕妈祖信仰中心;小公园中山纪念亭"},
			{kind: ActivitySight, time: "晚上", title: "海滨长廊", location: "汕头内海湾", notes: "看内海湾夜景 + 礐石大桥灯光"},
			{kind: ActivityLodging, time: "晚上", title: "入住汕头市区", location: "金平区 / 龙湖区", notes: "市区近小公园便于逛吃"},
			{kind: ActivityFood, time: "全天", title: "牛肉火锅 + 牛肉丸 + 粿品", location: "金平区 / 龙湖区", notes: "推荐杏花吴记 / 海记牛肉店;牛肉丸 Q 弹弹牙"},
		},
	},
	{
		date:    "2026-10-04",
		city:    "南澳岛",
		title:   "D11 汕头 → 南澳岛",
		summary: "南澳大桥 → 青澳湾 → 黄花山森林公园,晚上看日落 + 海鲜。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "汕头 → 南澳岛(过南澳大桥)", location: "南澳大桥", notes: "大桥 11 公里,自驾 / 包车方便"},
			{kind: ActivitySight, time: "上午", title: "青澳湾 + 北回归线广场", location: "青澳湾", notes: "北回归线标志塔「自然之门」打卡"},
			{kind: ActivitySight, time: "下午", title: "黄花山森林公园 + 风电场", location: "黄花山", notes: "风车山看大风车 + 海景,适合航拍"},
			{kind: ActivitySight, time: "晚上", title: "青澳湾看日落", location: "青澳湾海滩", notes: "沙滩细腻,可下水"},
			{kind: ActivityLodging, time: "晚上", title: "入住青澳湾 / 后宅镇", location: "南澳岛", notes: "岛上民宿为主,提前订"},
			{kind: ActivityFood, time: "晚上", title: "海鲜大餐", location: "青澳湾 / 后宅镇", notes: "紫菜炒饭 + 椒盐皮皮虾 + 扇贝粉丝"},
		},
	},
	{
		date:    "2026-10-05",
		city:    "潮州",
		title:   "D12 南澳 → 潮州",
		summary: "上午潮州古城 + 广济桥 + 韩文公祠,下午开元寺 + 牌坊街。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "南澳 → 潮州", location: "南澳大桥 → 潮州古城", notes: "约 1.5 小时车程"},
			{kind: ActivitySight, time: "上午", title: "潮州古城 + 广济桥 + 韩文公祠", location: "湘桥区", notes: "广济桥是四大古桥之一,韩文公祠为纪念韩愈"},
			{kind: ActivityFood, time: "中午", title: "潮州菜", location: "潮州古城", notes: "推荐官塘兄弟牛肉店 / 潮膳楼 —— 卤鹅 / 鱼生 / 蚝烙"},
			{kind: ActivitySight, time: "下午", title: "开元寺 + 牌坊街", location: "湘桥区", notes: "牌坊街 22 座明清石牌坊,古韵十足"},
			{kind: ActivitySight, time: "晚上", title: "牌坊街夜景 + 广济桥灯光秀", location: "湘桥区", notes: "广济桥每晚 20:00 灯光秀,免费观看"},
			{kind: ActivityLodging, time: "晚上", title: "入住潮州古城", location: "湘桥区", notes: "古城内民宿步行可达牌坊街"},
		},
	},
	{
		date:    "2026-10-06",
		city:    "揭阳",
		title:   "D13 潮州 → 揭阳 / 返程缓冲日",
		summary: "上午揭阳学宫或黄满寨瀑布,下午自由活动 / 收拾行李。",
		activities: []seedActivity{
			{kind: ActivityTransport, time: "上午", title: "潮州 → 揭阳", location: "潮州 → 揭阳", notes: "高铁约 15 分钟"},
			{kind: ActivitySight, time: "上午", title: "揭阳学宫 / 黄满寨瀑布", location: "揭阳市区 / 揭西", notes: "学宫看岭南最大孔庙;黄满寨瀑布群距市区 1.5h 需预留"},
			{kind: ActivityNote, time: "下午", title: "根据返程交通调整", location: "—", notes: "若次日早班机/高铁,下午打包;若晚班可再加一站"},
			{kind: ActivityShopping, time: "晚上", title: "自由活动 / 买伴手礼", location: "潮州 / 汕头", notes: "牛肉丸、潮州柑、潮汕三宝、老药桔"},
		},
	},
	{
		date:    "2026-10-07",
		city:    "散团",
		title:   "D14 散团返程",
		summary: "自由活动,集中买伴手礼后各自返程。",
		activities: []seedActivity{
			{kind: ActivityShopping, time: "上午", title: "购买伴手礼", location: "潮州 / 汕头 / 揭阳机场", notes: "潮汕牛肉丸、潮州柑、潮汕三宝、老药桔"},
			{kind: ActivityShopping, time: "上午", title: "澳门钜记饼家 + 香港美心月饼", location: "澳门 / 香港免税店", notes: "钜记猪扒包 / 杏仁饼;美心流心奶黄月饼"},
			{kind: ActivityNote, time: "下午", title: "各自返程", location: "揭阳潮汕机场 / 汕头站 / 高铁", notes: "揭阳机场航班较多,高铁去广州 / 深圳中转也行"},
		},
	},
}

// EnsureSeeded 是 CLI 启动时的便利入口:若 repo 为空就 seed,返回是否真 seed 过。
// 出错且不是 already-seeded 时回传 error。
func EnsureSeeded(ctx context.Context, svc *Service) (*Trip, bool, error) {
	if existing, err := svc.List(ctx); err == nil && len(existing) > 0 {
		loaded, err := svc.Show(ctx, existing[0].ID)
		if err != nil {
			return nil, false, err
		}
		return loaded, false, nil
	}
	return SeedFirstTrip(ctx, svc)
}

// ErrSeedSkipped —— 当用户显式 seed 但已存在同行程时返回。
var ErrSeedSkipped = errors.New("seed: trip already exists")
