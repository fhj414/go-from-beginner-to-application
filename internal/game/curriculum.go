// Package game 定义「小地鼠闯关」课程内容与关卡元数据（由 Go 服务下发给前端）。
package game

// LessonKind 决定前端用哪种小游戏组件渲染。
type LessonKind string

const (
	KindPickOne LessonKind = "pick_one" // 单选题
	KindOrder   LessonKind = "order"    // 排序：把步骤排成正确顺序
)

// Lesson 一关教学内容。
type Lesson struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Subtitle    string     `json:"subtitle"`
	Kind        LessonKind `json:"kind"`
	Story       string     `json:"story"`
	Question    string     `json:"question"`
	Options     []string   `json:"options,omitempty"`
	Correct     int        `json:"correct,omitempty"`      // pick_one: 0-based index
	OrderItems  []string   `json:"order_items,omitempty"`  // order: 乱序由前端 shuffle
	OrderAnswer []int      `json:"order_answer,omitempty"` // 正确顺序为 items 的下标排列
	Hint        string     `json:"hint"`
	RewardXP    int        `json:"reward_xp"`
	AnyChoice   bool       `json:"any_choice,omitempty"` // 为真时单选题任选项都算对（轻松关）
}

// Curriculum 全部课程。
type Curriculum struct {
	Title   string   `json:"title"`
	Tagline string   `json:"tagline"`
	Lessons []Lesson `json:"lessons"`
}

// DefaultCurriculum 面向零基础学习者的轻量 Go 入门关卡。
func DefaultCurriculum() Curriculum {
	return Curriculum{
		Title:   "小地鼠闯 Go 星球",
		Tagline: "点点玩玩，把 Go 的最小常识装进脑袋～",
		Lessons: []Lesson{
			{
				ID: 1, Title: "你好呀，Go！", Subtitle: "先认识新朋友",
				Kind:     KindPickOne,
				Story:    "Go 是一只很勤快的小地鼠，喜欢把程序编译成跑得飞快的机器码。下面哪一句最像它的自我介绍？",
				Question: "哪一句更像 Go 的特点？",
				Options: []string{
					"我主要在浏览器里直接解释执行",
					"我通常先编译，再运行，启动很快",
					"我只能用来做网页动画",
				},
				Correct:  1,
				Hint:     "想想：Go 是编译型语言哦。",
				RewardXP: 10,
			},
			{
				ID: 2, Title: "第一个程序", Subtitle: "经典开场白",
				Kind:     KindOrder,
				Story:    "想打印 “Hello, Go!” 需要几步？把下面卡片排成你觉得合理的顺序。",
				Question: "把步骤拖到正确顺序（点两下也可交换哦）",
				OrderItems: []string{
					"写 package main",
					"写 import \"fmt\"",
					"写 func main() { ... }",
					"在 main 里调用 fmt.Println(\"Hello, Go!\")",
				},
				OrderAnswer: []int{0, 1, 2, 3},
				Hint:        "包 → 引入工具 → 入口函数 → 打印。",
				RewardXP:    15,
			},
			{
				ID: 3, Title: "变量小盒子", Subtitle: ":= 与 var",
				Kind:     KindPickOne,
				Story:    "变量像贴标签的小盒子，里面装着数据。Go 里常用 := 在函数里快速声明。",
				Question: "下面哪种写法在函数体内最常见？",
				Options: []string{
					"name := \"地鼠\"",
					"name == \"地鼠\"",
					"name :: \"地鼠\"",
				},
				Correct:  0,
				Hint:     ":= 是声明并赋值。",
				RewardXP: 12,
			},
			{
				ID: 4, Title: "函数像乐高", Subtitle: "输入 → 输出",
				Kind:     KindPickOne,
				Story:    "函数把重复的事情打包。下面这个小函数想表达「两个整数相加」。",
				Question: "返回值类型写哪一个最合适？",
				Options: []string{
					"func add(a int, b int) string { return a + b }",
					"func add(a int, b int) int { return a + b }",
					"func add(a int, b int) { return a + b }",
				},
				Correct:  1,
				Hint:     "两个 int 相加，返回还是 int。",
				RewardXP: 14,
			},
			{
				ID: 5, Title: "错误也要说清楚", Subtitle: "error 值",
				Kind:     KindPickOne,
				Story:    "Go 不爱乱抛异常，更习惯把 error 当作返回值的一部分。",
				Question: "下面哪一句更符合 Go 的习惯？",
				Options: []string{
					"出错了就 panic，业务里到处用",
					"能处理就处理，必要时把 error 往上返回",
					"忽略 error，编译器会帮我兜底",
				},
				Correct:  1,
				Hint:     "生产代码里 panic 要克制。",
				RewardXP: 12,
			},
			{
				ID: 6, Title: "并发小魔法", Subtitle: "goroutine",
				Kind:     KindPickOne,
				Story:    "go f() 会启动一个 goroutine，像轻轻招一下手：‘这件事并行去做吧’。",
				Question: "启动 goroutine 的关键词是？",
				Options:  []string{"async", "go", "spawn"},
				Correct:  1,
				Hint:     "只有一个超短关键字。",
				RewardXP: 15,
			},
			{
				ID: 7, Title: "通道传纸条", Subtitle: "channel",
				Kind:     KindOrder,
				Story:    "channel 像两人之间传纸条的管道，帮 goroutine 之间安全地传递数据。",
				Question: "把概念卡排成更合理的解释顺序：",
				OrderItems: []string{
					"多个 goroutine 同时跑",
					"用 channel 发送数据",
					"另一端接收数据",
					"配合读写避免乱抢共享状态",
				},
				OrderAnswer: []int{0, 1, 2, 3},
				Hint:        "先并发，再通信，再协作。",
				RewardXP:    18,
			},
			{
				ID: 8, Title: "毕业小礼花", Subtitle: "你已经是初级探险家啦",
				Kind:     KindPickOne,
				Story:    "你已经了解了 Go 的几个核心气质：编译快、error 显式、并发用 goroutine + channel。",
				Question: "接下来最想做什么？（没有标准答案，选个最让你开心的）",
				Options: []string{
					"写一个小 HTTP 服务试试",
					"先休息，吃根胡萝卜",
					"把今天的笔记整理成一张小抄",
				},
				Correct:   0,
				AnyChoice: true,
				Hint:      "选什么都加经验——学习也要开心呀。",
				RewardXP:  20,
			},
			{
				ID: 9, Title: "小服务开张啦", Subtitle: "简单应用：HTTP",
				Kind:     KindPickOne,
				Story:    "如果你想做一个最简单的 Go Web 小服务，通常会先从标准库 net/http 开始，就像搭一个小小售票亭。",
				Question: "下面哪个包最常用来起一个基础 HTTP 服务？",
				Options: []string{
					`net/http`,
					`fmt/http`,
					`browser/go`,
				},
				Correct:  0,
				Hint:     "Go 标准库已经内置了 HTTP 能力。",
				RewardXP: 18,
			},
			{
				ID: 10, Title: "JSON 快递员", Subtitle: "简单应用：接口返回",
				Kind:     KindPickOne,
				Story:    "很多 Go 服务会把数据变成 JSON 再返回给前端，就像把礼物装进统一的小盒子里。",
				Question: "下面哪个标准库包最常用来处理 JSON？",
				Options: []string{
					`encoding/json`,
					`net/json`,
					`fmt/json`,
				},
				Correct:  0,
				Hint:     "名字里就写着 encoding 哦。",
				RewardXP: 18,
			},
		},
	}
}

// StageForProgress 根据完成关卡数返回阶段标识（前端可做不同装扮与动效）。
func StageForProgress(completed int, total int) string {
	if total <= 0 {
		return "seed"
	}
	p := float64(completed) / float64(total)
	switch {
	case p <= 0:
		return "seed"
	case p < 0.35:
		return "sprout"
	case p < 0.7:
		return "scout"
	case p < 1:
		return "ninja"
	default:
		return "star"
	}
}
