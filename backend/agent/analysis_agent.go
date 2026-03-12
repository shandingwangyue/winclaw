package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
	"winclaw/backend/ai"
	"winclaw/backend/utils"
)

// AnalysisAgent 数据分析Agent
type AnalysisAgent struct {
	aiClient *ai.OpenAIClient
}

// NewAnalysisAgent 创建数据分析Agent
func NewAnalysisAgent(aiClient *ai.OpenAIClient) *AnalysisAgent {
	return &AnalysisAgent{
		aiClient: aiClient,
	}
}

// Process 处理数据分析相关请求
func (ana *AnalysisAgent) Process(ctx context.Context, query string, params map[string]string) (string, error) {
	// 根据查询内容判断用户意图
	if utils.ContainsIgnoreCase(query, "分析") || utils.ContainsIgnoreCase(query, "统计") || utils.ContainsIgnoreCase(query, "报表") ||
		utils.ContainsIgnoreCase(query, "analyze") || utils.ContainsIgnoreCase(query, "report") || utils.ContainsIgnoreCase(query, "stats") {
		return ana.performAnalysis(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "趋势") || utils.ContainsIgnoreCase(query, "趋势图") || utils.ContainsIgnoreCase(query, "chart") ||
		utils.ContainsIgnoreCase(query, "graph") || utils.ContainsIgnoreCase(query, "visualization") {
		return ana.generateTrendChart(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "对比") || utils.ContainsIgnoreCase(query, "比较") || utils.ContainsIgnoreCase(query, "compare") ||
		utils.ContainsIgnoreCase(query, "vs") {
		return ana.compareData(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "预测") || utils.ContainsIgnoreCase(query, "forecast") || utils.ContainsIgnoreCase(query, "predict") {
		return ana.predictData(ctx, query, params)
	} else if utils.ContainsIgnoreCase(query, "摘要") || utils.ContainsIgnoreCase(query, "总结") || utils.ContainsIgnoreCase(query, "summary") ||
		utils.ContainsIgnoreCase(query, "overview") {
		return ana.generateSummary(ctx, query, params)
	} else {
		// 使用AI进行更复杂的意图理解
		return ana.processWithAI(ctx, query, params)
	}
}

// performAnalysis 执行数据分析
func (ana *AnalysisAgent) performAnalysis(ctx context.Context, query string, params map[string]string) (string, error) {
	dataType := params["data_type"]
	if dataType == "" {
		// 从查询中推断数据类型
		if utils.ContainsIgnoreCase(query, "销售") || utils.ContainsIgnoreCase(query, "sales") {
			dataType = "sales"
		} else if utils.ContainsIgnoreCase(query, "客户") || utils.ContainsIgnoreCase(query, "customer") {
			dataType = "customer"
		} else if utils.ContainsIgnoreCase(query, "产品") || utils.ContainsIgnoreCase(query, "product") {
			dataType = "product"
		} else {
			dataType = "general"
		}
	}

	timeRange := params["time_range"]
	if timeRange == "" {
		timeRange = "last_month" // 默认时间范围
	}

	// 模拟数据分析结果
	analysisResult := generateSampleData(dataType, timeRange)

	result := fmt.Sprintf("数据分析结果 (%s):\n\n", dataType)
	result += fmt.Sprintf("时间范围: %s\n", timeRange)
	result += fmt.Sprintf("数据概览:\n")
	result += fmt.Sprintf("- 总数: %d\n", analysisResult["total"])
	result += fmt.Sprintf("- 平均值: %.2f\n", analysisResult["average"])
	result += fmt.Sprintf("- 最大值: %.2f\n", analysisResult["max"])
	result += fmt.Sprintf("- 最小值: %.2f\n", analysisResult["min"])

	// 添加特定类型的分析
	switch dataType {
	case "sales":
		result += fmt.Sprintf("- 销售额增长: %.2f%%\n", analysisResult["growth_rate"])
		result += fmt.Sprintf("- 最佳销售产品: %s\n", analysisResult["top_product"])
	case "customer":
		result += fmt.Sprintf("- 新增客户: %d\n", analysisResult["new_customers"])
		result += fmt.Sprintf("- 客户满意度: %.2f%%\n", analysisResult["satisfaction"])
	case "product":
		result += fmt.Sprintf("- 最受欢迎产品: %s\n", analysisResult["top_product"])
		result += fmt.Sprintf("- 库存周转率: %.2f\n", analysisResult["turnover_rate"])
	}

	return result, nil
}

// generateTrendChart 生成趋势图表
func (ana *AnalysisAgent) generateTrendChart(ctx context.Context, query string, params map[string]string) (string, error) {
	dataType := params["data_type"]
	if dataType == "" {
		dataType = "sales"
	}

	timeRange := params["time_range"]
	if timeRange == "" {
		timeRange = "last_quarter"
	}

	// 生成简单的文本图表
	dataPoints := generateTimeSeriesData(dataType, timeRange)

	result := fmt.Sprintf("趋势分析 (%s - %s):\n\n", dataType, timeRange)

	// 创建简单的文本图表
	maxValue := 0.0
	for _, point := range dataPoints {
		if point.Value > maxValue {
			maxValue = point.Value
		}
	}

	if maxValue == 0 {
		maxValue = 1 // 避免除零错误
	}

	for _, point := range dataPoints {
		barLength := int((point.Value / maxValue) * 30) // 最大长度为30个字符
		bar := strings.Repeat("█", barLength)
		result += fmt.Sprintf("%-10s |%s %.2f\n", point.Label, bar, point.Value)
	}

	return result, nil
}

// compareData 对比数据
func (ana *AnalysisAgent) compareData(ctx context.Context, query string, params map[string]string) (string, error) {
	category1 := params["category1"]
	category2 := params["category2"]

	if category1 == "" || category2 == "" {
		// 从查询中提取对比类别
		parts := strings.Split(query, "vs")
		if len(parts) >= 2 {
			category1 = strings.TrimSpace(parts[0])
			category2 = strings.TrimSpace(parts[1])
		} else {
			// 默认对比
			category1 = "去年"
			category2 = "今年"
		}
	}

	dataType := params["data_type"]
	if dataType == "" {
		dataType = "sales"
	}

	// 生成对比数据
	value1 := calculateComparisonValue(category1, dataType)
	value2 := calculateComparisonValue(category2, dataType)

	difference := value2 - value1
	growthRate := 0.0
	if value1 != 0 {
		growthRate = (difference / value1) * 100
	}

	result := fmt.Sprintf("数据对比分析:\n")
	result += fmt.Sprintf("%s: %.2f\n", category1, value1)
	result += fmt.Sprintf("%s: %.2f\n", category2, value2)
	result += fmt.Sprintf("差异: %.2f\n", difference)
	result += fmt.Sprintf("增长率: %.2f%%\n", growthRate)

	if growthRate > 0 {
		result += fmt.Sprintf("结论: %s 表现优于 %s\n", category2, category1)
	} else {
		result += fmt.Sprintf("结论: %s 表现优于 %s\n", category1, category2)
	}

	return result, nil
}

// predictData 预测数据
func (ana *AnalysisAgent) predictData(ctx context.Context, query string, params map[string]string) (string, error) {
	dataType := params["data_type"]
	if dataType == "" {
		dataType = "sales"
	}

	duration := params["duration"]
	if duration == "" {
		duration = "next_month"
	}

	// 基于历史数据进行简单预测
	historicalData := generateHistoricalData(dataType)
	prediction := calculatePrediction(historicalData)

	result := fmt.Sprintf("数据预测 (%s - %s):\n", dataType, duration)
	result += fmt.Sprintf("预测值: %.2f\n", prediction)
	result += fmt.Sprintf("置信度: 85%%\n")
	result += fmt.Sprintf("影响因素: 市场趋势、季节性、历史表现\n")

	return result, nil
}

// generateSummary 生成数据摘要
func (ana *AnalysisAgent) generateSummary(ctx context.Context, query string, params map[string]string) (string, error) {
	dataType := params["data_type"]
	if dataType == "" {
		dataType = "general"
	}

	timeRange := params["time_range"]
	if timeRange == "" {
		timeRange = "this_month"
	}

	// 生成数据摘要
	summary := generateDataSummary(dataType, timeRange)

	result := fmt.Sprintf("数据摘要 (%s - %s):\n\n", dataType, timeRange)
	result += summary

	return result, nil
}

// processWithAI 使用AI处理复杂数据分析请求
func (ana *AnalysisAgent) processWithAI(ctx context.Context, query string, params map[string]string) (string, error) {
	systemPrompt := `你是一个专业的数据分析助手，能够帮助用户处理各种数据分析任务。

你可以协助用户：
1. 执行各类数据分析（销售、客户、产品等）
2. 生成趋势图表和可视化
3. 进行数据对比分析
4. 预测未来趋势
5. 生成数据摘要和报告
6. 解释数据指标和业务含义

请根据用户的具体需求提供相应的帮助，并以清晰易懂的方式呈现分析结果。`

	response, err := ana.aiClient.Chat(systemPrompt, query)
	if err != nil {
		return "", err
	}

	return response, nil
}

// 辅助函数：生成示例数据
func generateSampleData(dataType, timeRange string) map[string]interface{} {
	result := make(map[string]interface{})

	// 根据不同类型返回不同的示例数据
	switch dataType {
	case "sales":
		result["total"] = 150
		result["average"] = 1250.50
		result["max"] = 5000.00
		result["min"] = 100.00
		result["growth_rate"] = 15.5
		result["top_product"] = "产品A"
	case "customer":
		result["total"] = 200
		result["average"] = 0
		result["max"] = 0
		result["min"] = 0
		result["new_customers"] = 25
		result["satisfaction"] = 92.3
	case "product":
		result["total"] = 50
		result["average"] = 0
		result["max"] = 0
		result["min"] = 0
		result["top_product"] = "产品C"
		result["turnover_rate"] = 2.5
	default:
		result["total"] = 100
		result["average"] = 500.00
		result["max"] = 2000.00
		result["min"] = 50.00
	}

	return result
}

// 时间序列数据点结构
type TimeSeriesPoint struct {
	Label string
	Value float64
}

// 生成时间序列数据
func generateTimeSeriesData(dataType, timeRange string) []TimeSeriesPoint {
	points := []TimeSeriesPoint{}

	// 根据时间范围生成数据点
	labels := []string{"1月", "2月", "3月", "4月", "5月", "6月"}
	if timeRange == "last_quarter" {
		labels = []string{"4月", "5月", "6月"}
	} else if timeRange == "last_year" {
		labels = []string{"Q1", "Q2", "Q3", "Q4"}
	}

	baseValue := 1000.0
	if dataType == "sales" {
		baseValue = 5000.0
	} else if dataType == "customer" {
		baseValue = 200.0
	}

	for i, label := range labels {
		value := baseValue + float64(i)*100 + randFloat()
		points = append(points, TimeSeriesPoint{Label: label, Value: value})
	}

	return points
}

// 计算对比值
func calculateComparisonValue(category, dataType string) float64 {
	baseValue := 1000.0
	if dataType == "sales" {
		baseValue = 5000.0
	} else if dataType == "customer" {
		baseValue = 200.0
	}

	// 根据类别调整值
	if utils.ContainsIgnoreCase(category, "今年") || utils.ContainsIgnoreCase(category, "next") {
		return baseValue * 1.15 // 假设增长15%
	} else if utils.ContainsIgnoreCase(category, "去年") || utils.ContainsIgnoreCase(category, "last") {
		return baseValue
	}

	return baseValue + randFloat()*200
}

// 生成历史数据
func generateHistoricalData(dataType string) []float64 {
	data := []float64{}
	baseValue := 1000.0
	if dataType == "sales" {
		baseValue = 5000.0
	} else if dataType == "customer" {
		baseValue = 200.0
	}

	for i := 0; i < 12; i++ {
		value := baseValue + float64(i)*50 + randFloat()*100
		data = append(data, value)
	}

	return data
}

// 计算预测值
func calculatePrediction(history []float64) float64 {
	if len(history) == 0 {
		return 0
	}

	// 简单的线性回归预测
	sum := 0.0
	for _, v := range history {
		sum += v
	}
	avg := sum / float64(len(history))

	// 假设增长趋势
	lastValue := history[len(history)-1]
	predicted := (avg + lastValue) / 2 * 1.05 // 假设5%增长

	return predicted
}

// 生成数据摘要
func generateDataSummary(dataType, timeRange string) string {
	summary := ""

	switch dataType {
	case "sales":
		summary = `关键指标:
- 总销售额: ¥750,000
- 订单数量: 150
- 平均订单价值: ¥5,000
- 最佳销售月份: 3月
- 主要销售渠道: 线上

主要发现:
- 本月销售额环比增长12%
- 新客户贡献了30%的销售额
- 产品A是最畅销的产品
- 客户回购率为65%`
	case "customer":
		summary = `客户概况:
- 总客户数: 1,200
- 新增客户: 45
- 客户满意度: 92.5%
- 客户流失率: 3.2%

主要洞察:
- 高价值客户占比提升至25%
- 客户平均生命周期价值增长8%
- 主要客户群体为25-40岁
- 客户支持响应时间平均2小时`
	case "product":
		summary = `产品表现:
- 在售产品: 50种
- 畅销产品: 15种
- 库存周转率: 2.8次/年
- 产品退货率: 1.5%

重要趋势:
- 产品A需求增长显著
- 季节性产品销售符合预期
- 新品上市成功率80%
- 产品利润率稳定在35%`
	default:
		summary = `数据摘要:
- 数据完整性: 98%
- 更新频率: 实时
- 数据源: 多渠道集成
- 分析维度: 时间、类别、地区

总体评估:
- 数据质量良好
- 趋势符合预期
- 异常值已标识
- 建议关注重点指标`
	}

	return summary
}

// 生成随机浮点数辅助函数
func randFloat() float64 {
	// 简单的伪随机数生成，实际应用中应使用 math/rand 包
	now := time.Now().UnixNano()
	return float64(now%1000) / 100.0
}
