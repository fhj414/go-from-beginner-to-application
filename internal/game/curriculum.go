// Package game 定义「小地鼠闯关」课程内容与关卡元数据（由 Go 服务下发给前端）。
package game

import (
	"fmt"
	"strings"
)

// LessonKind 决定前端用哪种小游戏组件渲染。
type LessonKind string

const (
	KindPickOne  LessonKind = "pick_one"   // 单选题
	KindOrder    LessonKind = "order"      // 排序：把步骤排成正确顺序
	KindFillText LessonKind = "fill_text"  // 填空：输入简短关键字/代码片段
	KindPair     LessonKind = "pair_match" // 配对：把左右概念连起来
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
	Correct     int        `json:"correct"`                // pick_one: 0-based index
	OrderItems  []string   `json:"order_items,omitempty"`  // order: 乱序由前端 shuffle
	OrderAnswer []int      `json:"order_answer,omitempty"` // 正确顺序为 items 的下标排列
	FillAnswer  string     `json:"fill_answer,omitempty"`
	FillAliases []string   `json:"fill_aliases,omitempty"`
	FillPrefix  string     `json:"fill_prefix,omitempty"`
	FillSuffix  string     `json:"fill_suffix,omitempty"`
	PairLeft    []string   `json:"pair_left,omitempty"`
	PairRight   []string   `json:"pair_right,omitempty"`
	PairAnswer  []int      `json:"pair_answer,omitempty"` // left index -> right index
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

func pick(id int, title, subtitle, story, question string, options []string, correct int, hint string, xp int) Lesson {
	return Lesson{
		ID:       id,
		Title:    title,
		Subtitle: subtitle,
		Kind:     KindPickOne,
		Story:    story,
		Question: question,
		Options:  options,
		Correct:  correct,
		Hint:     hint,
		RewardXP: xp,
	}
}

func order(id int, title, subtitle, story, question string, items []string, answer []int, hint string, xp int) Lesson {
	return Lesson{
		ID:          id,
		Title:       title,
		Subtitle:    subtitle,
		Kind:        KindOrder,
		Story:       story,
		Question:    question,
		OrderItems:  items,
		OrderAnswer: answer,
		Hint:        hint,
		RewardXP:    xp,
	}
}

func fill(id int, title, subtitle, story, question, prefix, suffix, answer string, aliases []string, hint string, xp int) Lesson {
	return Lesson{
		ID:          id,
		Title:       title,
		Subtitle:    subtitle,
		Kind:        KindFillText,
		Story:       story,
		Question:    question,
		FillPrefix:  prefix,
		FillSuffix:  suffix,
		FillAnswer:  answer,
		FillAliases: aliases,
		Hint:        hint,
		RewardXP:    xp,
	}
}

func pair(id int, title, subtitle, story, question string, left, right []string, answer []int, hint string, xp int) Lesson {
	return Lesson{
		ID:         id,
		Title:      title,
		Subtitle:   subtitle,
		Kind:       KindPair,
		Story:      story,
		Question:   question,
		PairLeft:   left,
		PairRight:  right,
		PairAnswer: answer,
		Hint:       hint,
		RewardXP:   xp,
	}
}

// DefaultCurriculum 面向零基础学习者的第一章完整版。
func DefaultCurriculum() Curriculum {
	lessons := []Lesson{
		pick(1, "你好呀，Go！", "认识新朋友", "Go 像一只做事很利落的小地鼠，最擅长把程序先编译好再飞快跑起来。", "哪一句更像 Go 的特点？", []string{
			"我主要在浏览器里直接解释执行",
			"我通常先编译，再运行，启动很快",
			"我只能用来做网页动画",
		}, 1, "想想：Go 是编译型语言哦。", 10),
		order(2, "第一个程序", "经典开场白", "想打印 “Hello, Go!” 需要几步？把卡片排成顺序。", "把步骤拖到正确顺序（点两下也可交换）", []string{
			"写 package main",
			"写 import \"fmt\"",
			"写 func main() { ... }",
			"在 main 里调用 fmt.Println(\"Hello, Go!\")",
		}, []int{0, 1, 2, 3}, "包 → 引入工具 → 入口函数 → 打印。", 15),
		fill(3, "补上包名", "最小程序", "每个可运行 Go 程序，通常都从同一个包名开始。", "把空格补完整", "package ", "", "main", []string{"main"}, "能直接运行的入口包一般叫 main。", 12),
		pick(4, "变量小盒子", ":= 与 var", "变量像贴标签的小盒子，里面装着数据。", "下面哪种写法在函数体内最常见？", []string{
			"name := \"地鼠\"",
			"name == \"地鼠\"",
			"name :: \"地鼠\"",
		}, 0, ":= 是声明并赋值。", 12),
		fill(5, "打印函数", "fmt 小工具", "fmt 包像会说话的小喇叭，最常见的打印函数就是它。", "补上调用名", "fmt.", "(\"Hello\")", "Println", []string{"println", "Println"}, "换行打印最常见。", 12),
		pick(6, "数字还是文字", "基础类型", "Go 很在意数据类型，数字和字符串要分清。", "下面哪个是字符串？", []string{
			"42",
			"\"42\"",
			"3.14",
		}, 1, "带双引号的是字符串。", 10),
		pick(7, "函数像乐高", "输入 → 输出", "函数把重复的事情打包。", "返回值类型写哪一个最合适？", []string{
			"func add(a int, b int) string { return a + b }",
			"func add(a int, b int) int { return a + b }",
			"func add(a int, b int) { return a + b }",
		}, 1, "两个 int 相加，返回还是 int。", 14),
		fill(8, "返回关键字", "把结果交出去", "函数算出结果后，要用一个关键字把它交出去。", "把空格补完整", "", " total", "return", []string{"return"}, "这个词很常见。", 12),
		pick(9, "if 小门卫", "条件判断", "if 就像门卫，条件对了才放行。", "下面哪个最像 if 的用途？", []string{
			"重复执行很多次",
			"根据条件决定走哪条路",
			"定义一个新类型",
		}, 1, "条件判断就是 if 的强项。", 12),
		order(10, "for 小转盘", "循环", "Go 没有 while，很多循环都靠 for。", "把 for 循环的理解顺序排一排", []string{
			"先设置起点",
			"检查条件是否继续",
			"执行循环体",
			"更新计数器",
		}, []int{0, 1, 2, 3}, "起点 → 条件 → 执行 → 更新。", 15),
		pick(11, "数组与切片", "一排小格子", "数组长度固定，切片更灵活，所以日常开发更爱用切片。", "哪一个更灵活？", []string{
			"数组 array",
			"切片 slice",
			"都完全一样",
		}, 1, "日常开发里，更常用 slice 来处理列表。", 13),
		fill(12, "append 出场", "给切片加东西", "切片想再装一个元素时，最常见的伙伴就是 append。", "补上函数名", "", "(items, 3)", "append", []string{"append"}, "往 slice 里追加内容。", 13),
		pick(13, "map 像字典", "键值对", "map 很适合存“名字 -> 值”的关系。", "下面哪个场景最适合 map？", []string{
			"保存学生姓名对应分数",
			"只保存一个整数",
			"表示固定长度的三张卡片",
		}, 0, "键值对就是 map 的主场。", 13),
		fill(14, "取长度", "len", "不论是字符串、切片还是 map，经常都要看看有多长。", "补上函数名", "", "(names)", "len", []string{"len"}, "长度函数只有 3 个字母。", 12),
		pick(15, "struct 小积木", "组织数据", "struct 就像把一组相关信息绑在一起的小盒子。", "什么时候适合用 struct？", []string{
			"只保存一个布尔值",
			"想把姓名、年龄、城市放成一个整体",
			"想做网络请求",
		}, 1, "多个相关字段放一起。", 14),
		fill(16, "字段访问", "点一下", "有了 user.Name 这种写法，就能拿到结构体里的字段。", "补上点号后的字段名", "user.", "", "Name", []string{"name", "Name"}, "结构体字段常用大写字母开头。", 12),
		pick(17, "方法是什么", "给类型加本领", "方法像是“这个类型自带的动作”。", "下面哪种理解更接近方法？", []string{
			"附着在类型上的函数",
			"只能在 main 里写的代码",
			"变量的别名",
		}, 0, "方法 = 类型 + 行为。", 14),
		pick(18, "指针轻触一下", "共享同一个对象", "指针不是魔法，本质上是“指向某个值的位置”。", "什么时候更可能用指针接收者？", []string{
			"想修改原对象内容",
			"只是打印一个常量",
			"只想做字符串拼接",
		}, 0, "要改原值时常用指针。", 15),
		pick(19, "接口的气质", "interface", "Go 的 interface 更像“只要求你会做这些动作”，不用显式说自己实现了谁。", "哪句最符合 Go 的接口风格？", []string{
			"接口通常是小而专注的",
			"接口越大越好",
			"必须写 extends 才算实现",
		}, 0, "小接口更灵活。", 16),
		fill(20, "错误值", "error", "Go 不太爱乱抛异常，更多是把错误当成一个返回值。", "补上错误类型名", "", "", "error", []string{"error"}, "就是平时 if err != nil 里的那个。", 15),
		pick(21, "错误也要说清楚", "处理 error", "能处理就处理，不能处理就往上交。", "下面哪一句更符合 Go 的习惯？", []string{
			"出错了就 panic，业务里到处用",
			"能处理就处理，必要时把 error 往上返回",
			"忽略 error，编译器会帮我兜底",
		}, 1, "panic 要克制，error 要尊重。", 15),
		pick(22, "defer 收尾", "最后再做", "defer 像给未来留一张便签：函数结束前别忘了做这件事。", "下面哪个场景最常见？", []string{
			"关闭文件或释放资源",
			"声明变量类型",
			"定义结构体字段",
		}, 0, "close / unlock 这类清理动作最常见。", 15),
		pick(23, "并发小魔法", "goroutine", "go f() 会启动一个 goroutine，像轻轻招一下手：‘这件事并行去做吧’。", "启动 goroutine 的关键词是？", []string{
			"async",
			"go",
			"spawn",
		}, 1, "只有一个超短关键字。", 16),
		pick(24, "通道传纸条", "channel", "channel 像 goroutine 之间的安全纸条管道。", "channel 更像下面哪一种？", []string{
			"共享一堆全局变量",
			"协程之间传数据的管道",
			"页面里的按钮组件",
		}, 1, "关键是通信。", 16),
		order(25, "并发协作", "先并发再通信", "把并发协作的节奏整理顺。", "把概念卡排成更合理的解释顺序", []string{
			"多个 goroutine 同时跑",
			"用 channel 发送数据",
			"另一端接收数据",
			"协作完成任务",
		}, []int{0, 1, 2, 3}, "先并发，再通信，再完成。", 18),
		pick(26, "context 小遥控器", "取消任务", "当请求超时或用户离开时，context 能帮你通知下游别再忙了。", "context 最常用来做什么？", []string{
			"取消和控制任务生命周期",
			"定义数据库表",
			"替代 for 循环",
		}, 0, "它是取消和超时管理好帮手。", 17),
		pick(27, "包与导出", "大小写有意义", "Go 里标识符首字母大写通常表示可被包外访问。", "下面哪个字段更可能被包外访问？", []string{
			"name",
			"Name",
			"_name",
		}, 1, "首字母大写通常就是导出。", 14),
		fill(28, "格式化习惯", "写完就整理", "Go 社区有一个几乎人人都会用的代码格式化工具。", "补上命令名", "go ", "", "fmt", []string{"fmt"}, "写完 Go 代码常会先跑一下它。", 14),
		pick(29, "模块管理", "go.mod", "go.mod 像项目的说明书，告诉工具链这是个什么模块。", "哪一个文件最常记录模块路径和依赖？", []string{
			"go.mod",
			"main.sum",
			"package.lock",
		}, 0, "模块文件就是它。", 14),
		pick(30, "HTTP 服务", "最小接口", "如果你想做一个最简单的 Go Web 服务，通常会先从标准库 net/http 开始。", "哪个包最常用来起基础 HTTP 服务？", []string{
			"net/http",
			"fmt/http",
			"browser/go",
		}, 0, "标准库已经自带 HTTP。", 18),
		fill(31, "状态码", "接口回应", "HTTP 服务经常要回应一个状态码，表示请求是否成功。", "把空格补完整", "http.Status", "", "OK", []string{"ok", "OK"}, "成功状态码常量连起来就是 http.StatusOK。", 15),
		pick(32, "JSON 快递员", "接口返回", "很多 Go 服务会把数据编码成 JSON 再返回给前端。", "哪个标准库包最常用来处理 JSON？", []string{
			"encoding/json",
			"net/json",
			"fmt/json",
		}, 0, "名字里就写着 encoding。", 18),
		fill(33, "编码动作", "把数据变 JSON", "当你想把结构体编码成 JSON 时，常见方法名字是 Encode。", "补上方法名", "json.NewEncoder(w).", "(data)", "Encode", []string{"encode", "Encode"}, "不是 Marshal 这次。", 16),
		pick(34, "登录系统", "身份识别", "做登录系统时，服务端通常要知道“你是谁”。", "下面哪个更像登录系统要解决的问题？", []string{
			"识别当前请求对应哪个用户",
			"让颜色主题更好看",
			"减少 CSS 文件数量",
		}, 0, "登录本质是身份识别。", 17),
		pick(35, "缓存想法", "先从快的地方拿", "缓存像一个更近的小抽屉，先看看它有没有数据。", "缓存最常见的目标是什么？", []string{
			"减少重复计算或重复查询",
			"替代所有数据库",
			"让变量名字更短",
		}, 0, "先快拿，减少主链路压力。", 17),
		pair(36, "概念对对碰", "核心词汇配对", "现在来把常见概念和它们的职责对起来。", "把左边概念和右边描述配成一对", []string{
			"slice",
			"map",
			"struct",
		}, []string{
			"键值对容器",
			"可变长度序列",
			"把多字段组织成一个整体",
		}, []int{1, 0, 2}, "一个像列表，一个像字典，一个像打包数据的盒子。", 18),
		pair(37, "并发搭档", "goroutine / channel / context", "三个并发高频词，一口气分清。", "把概念和作用配起来", []string{
			"goroutine",
			"channel",
			"context",
		}, []string{
			"取消或控制任务生命周期",
			"轻量并发执行单元",
			"goroutine 之间传递数据",
		}, []int{1, 2, 0}, "谁负责跑、谁负责传、谁负责停。", 18),
		fill(38, "启动服务", "监听端口", "HTTP 服务启动时，经常会看到一个很常见的标准库函数。", "补上函数名", "http.", "(\":8080\", nil)", "ListenAndServe", []string{"listenandserve", "ListenAndServe"}, "它字面意思就是“监听并服务”。", 18),
		pick(39, "测试意识", "go test", "写 Go 不只是跑起来，也要学会验证。", "最常用的基础测试命令是哪一个？", []string{
			"go check",
			"go test",
			"go verify",
		}, 1, "Go 的测试命令非常直接。", 16),
		fill(40, "测试函数名", "Test 开头", "Go 的测试函数通常有固定命名习惯。", "补上函数名前缀", "", "Add(t *testing.T)", "Test", []string{"test", "Test"}, "测试函数通常从它开始。", 16),
		pick(41, "日志有什么用", "观察程序", "日志像程序留给你的面包屑，出问题时能帮你回头看。", "日志最核心的价值更接近哪一个？", []string{
			"帮助排查问题和理解程序运行过程",
			"让页面颜色更丰富",
			"替代所有注释",
		}, 0, "核心是可观察性。", 16),
		pair(42, "工程目录感", "cmd / internal / data", "项目目录开始变多时，知道大概放哪里会很重要。", "把目录和职责配起来", []string{
			"cmd/",
			"internal/",
			"data/",
		}, []string{
			"当前项目自己的业务代码",
			"程序入口 main",
			"本地持久化或样例数据",
		}, []int{1, 0, 2}, "入口、内部代码、数据文件，分开会更清楚。", 18),
		pick(43, "数据库连接", "先拿连接", "Go 连数据库时，常见做法不是每次手搓 socket，而是通过统一库管理连接。", "下面哪个标准库包最常见地和数据库编程一起出现？", []string{
			"database/sql",
			"storage/db",
			"db/core",
		}, 0, "Go 标准库里已经有数据库抽象层。", 18),
		pick(44, "接口设计感", "返回结构", "做 API 时，返回结构尽量稳定、清晰，会让前端更容易接。", "下面哪个更像好的接口返回习惯？", []string{
			"字段命名稳定、层次清楚",
			"每次想到什么就随便变结构",
			"成功失败都返回纯字符串",
		}, 0, "一致性很重要。", 17),
		fill(45, "环境变量", "配置入口", "部署时我们经常不用把密钥写死在代码里，而是从环境变量读取。", "补上变量名", "os.Getenv(\"", "\")", "PORT", []string{"port", "PORT"}, "这一关填的是示例里的变量名。", 17),
		pair(46, "部署小准备", "上线前的常见检查", "上线前别慌，先把几个常见动作想清楚。", "把动作和目的配起来", []string{
			"构建二进制",
			"设置环境变量",
			"跑测试",
		}, []string{
			"确认主要逻辑没有明显回归",
			"给程序提供端口或密钥等配置",
			"得到可运行产物",
		}, []int{2, 1, 0}, "产物、配置、验证，缺一不可。", 18),
		pick(47, "复盘思维", "学习不是只过题", "真正学会 Go，不只是答对题，还要能自己讲出来、做出来。", "下面哪个最能帮助你巩固？", []string{
			"把今天做过的小例子再手打一遍",
			"只截图，不再动手",
			"一直刷同一题，不看整体",
		}, 0, "动手复现最稳。", 16),
		pick(48, "毕业礼花 Max", "第一章完整版通关", "你已经把第一章真正走完了：基础、核心、并发、工程和简单应用都摸过一遍。", "接下来最值得做什么？", []string{
			"自己写一个小 HTTP API",
			"复习今天拿到星星较少的关卡",
			"邀请朋友一起玩再刷一遍",
		}, 0, "其实这三个都不错，继续动手最重要。", 26),
	}
	lessons[len(lessons)-1].AnyChoice = true
	return Curriculum{
		Title:   "小地鼠闯 Go 星球",
		Tagline: "不只是 10 个选择题，而是一整章真正能玩能学的 Go 入门冒险",
		Lessons: lessons,
	}
}

// NormalizeFillAnswer 供前端 / 测试参考。
func NormalizeFillAnswer(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidateCurriculum 在启动或测试时帮助发现题库配置错误。
func ValidateCurriculum(c Curriculum) error {
	if len(c.Lessons) == 0 {
		return fmt.Errorf("curriculum has no lessons")
	}
	seenIDs := map[int]bool{}
	for i, lesson := range c.Lessons {
		if lesson.ID <= 0 {
			return fmt.Errorf("lesson[%d] has invalid id %d", i, lesson.ID)
		}
		if seenIDs[lesson.ID] {
			return fmt.Errorf("lesson id %d duplicated", lesson.ID)
		}
		seenIDs[lesson.ID] = true
		if strings.TrimSpace(lesson.Title) == "" {
			return fmt.Errorf("lesson %d missing title", lesson.ID)
		}
		if strings.TrimSpace(lesson.Question) == "" {
			return fmt.Errorf("lesson %d missing question", lesson.ID)
		}
		switch lesson.Kind {
		case KindPickOne:
			if len(lesson.Options) < 2 {
				return fmt.Errorf("lesson %d pick_one needs at least 2 options", lesson.ID)
			}
			if lesson.Correct < 0 || lesson.Correct >= len(lesson.Options) {
				return fmt.Errorf("lesson %d pick_one correct index out of range", lesson.ID)
			}
		case KindOrder:
			if len(lesson.OrderItems) == 0 || len(lesson.OrderItems) != len(lesson.OrderAnswer) {
				return fmt.Errorf("lesson %d order config length mismatch", lesson.ID)
			}
			used := map[int]bool{}
			for _, idx := range lesson.OrderAnswer {
				if idx < 0 || idx >= len(lesson.OrderItems) {
					return fmt.Errorf("lesson %d order answer index out of range", lesson.ID)
				}
				if used[idx] {
					return fmt.Errorf("lesson %d order answer repeats index %d", lesson.ID, idx)
				}
				used[idx] = true
			}
		case KindFillText:
			if strings.TrimSpace(lesson.FillAnswer) == "" {
				return fmt.Errorf("lesson %d fill_text missing answer", lesson.ID)
			}
		case KindPair:
			if len(lesson.PairLeft) == 0 || len(lesson.PairLeft) != len(lesson.PairAnswer) {
				return fmt.Errorf("lesson %d pair_match config length mismatch", lesson.ID)
			}
			if len(lesson.PairRight) != len(lesson.PairLeft) {
				return fmt.Errorf("lesson %d pair_match left/right length mismatch", lesson.ID)
			}
			used := map[int]bool{}
			for _, idx := range lesson.PairAnswer {
				if idx < 0 || idx >= len(lesson.PairRight) {
					return fmt.Errorf("lesson %d pair answer index out of range", lesson.ID)
				}
				if used[idx] {
					return fmt.Errorf("lesson %d pair answer repeats index %d", lesson.ID, idx)
				}
				used[idx] = true
			}
		default:
			return fmt.Errorf("lesson %d has unknown kind %q", lesson.ID, lesson.Kind)
		}
	}
	return nil
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
	case p < 0.25:
		return "sprout"
	case p < 0.6:
		return "scout"
	case p < 1:
		return "ninja"
	default:
		return "star"
	}
}
