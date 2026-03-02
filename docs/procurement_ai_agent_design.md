# 采购 AI 智能体设计方案

## 一、整体架构

```
用户层
├── 采购员
└── 供应商

应用层
├── Web Portal
├── 移动端 App
└── API Gateway

AI 智能体层
├── 供应商画像 Agent
├── 资质审核 Agent
├── 供应商推荐 Agent
├── 报价分析 Agent
├── 比价 Agent
├── 流程自动化 Agent
└── 评标 Agent

核心服务层
├── 供应商服务
├── 采购服务
├── 报价服务
└── 流程服务

数据层
├── 业务数据库
├── 向量数据库
├── 知识图谱
└── 外部数据源
```

## 二、核心模块设计

### 1. 供应商画像 Agent

#### 功能
- 多维度数据采集与整合
- 实时画像更新
- 风险预警

#### 数据维度

| 维度 | 数据来源 | 更新频率 |
|------|----------|----------|
| 工商信息 | 工商局 API | 每日 |
| 信用评分 | 征信机构 | 每周 |
| 履约记录 | 内部系统 | 实时 |
| 质量指标 | 质检系统 | 实时 |
| ESG 评分 | 第三方 ESG 机构 | 每月 |
| 风险评估 | 风控模型 | 实时 |
| 舆情监控 | 新闻/社交媒体 | 每小时 |

#### 技术实现

```python
class SupplierProfileAgent:
    def __init__(self):
        self.data_collectors = {
            'business': BusinessDataCollector(),
            'credit': CreditDataCollector(),
            'performance': PerformanceDataCollector(),
            'quality': QualityDataCollector(),
            'esg': ESGDataCollector(),
            'risk': RiskAssessmentModel(),
            'sentiment': SentimentAnalyzer()
        }
        self.vector_db = VectorDatabase()
        self.knowledge_graph = KnowledgeGraph()
    
    def build_profile(self, supplier_id: str) -> SupplierProfile:
        """构建供应商 360 画像"""
        profile = SupplierProfile(supplier_id=supplier_id)
        
        # 并行采集多维度数据
        with ThreadPoolExecutor() as executor:
            futures = {
                executor.submit(collector.collect, supplier_id): dimension
                for dimension, collector in self.data_collectors.items()
            }
            
            for future in as_completed(futures):
                dimension = futures[future]
                profile.add_dimension(dimension, future.result())
        
        # 计算综合评分
        profile.calculate_overall_score()
        
        # 存储到向量数据库（用于语义搜索）
        self.vector_db.upsert(profile.to_vector())
        
        # 更新知识图谱
        self.knowledge_graph.update_node(profile)
        
        return profile
    
    def similar_suppliers(self, supplier_id: str, top_k: int = 10) -> List[Supplier]:
        """查找相似供应商"""
        profile = self.vector_db.get(supplier_id)
        return self.vector_db.search(profile.vector, top_k=top_k)
```

### 2. 资质审核 Agent

#### 功能
- 证照 OCR 识别
- 资质文件解析
- 合规性校验
- 过期预警

#### 技术实现

```python
class QualificationAgent:
    def __init__(self):
        self.ocr_engine = OCREngine()  # PaddleOCR / Tesseract
        self.document_parser = DocumentParser()
        self.compliance_checker = ComplianceChecker()
        self.llm = LLMClient()  # 用于复杂资质理解
    
    def review_qualifications(self, files: List[File]) -> ReviewResult:
        """审核供应商资质"""
        results = []
        
        for file in files:
            # OCR 识别
            text = self.ocr_engine.recognize(file)
            
            # 结构化解析
            parsed = self.document_parser.parse(text, file.type)
            
            # 合规性检查
            compliance = self.compliance_checker.check(parsed)
            
            # LLM 辅助理解复杂条款
            if parsed.has_complex_terms():
                analysis = self.llm.analyze(
                    prompt=f"分析以下资质文件的合规性：{text}",
                    context=self.get_compliance_rules()
                )
                compliance.update(analysis)
            
            results.append({
                'file': file.name,
                'parsed': parsed,
                'compliance': compliance,
                'risk_level': self.calculate_risk(compliance)
            })
        
        return ReviewResult(results)
    
    def check_expiry(self, supplier_id: str) -> List[Alert]:
        """检查资质过期"""
        qualifications = self.get_qualifications(supplier_id)
        alerts = []
        
        for qual in qualifications:
            if qual.is_expiring_soon(days=30):
                alerts.append(Alert(
                    type='EXPIRY_WARNING',
                    message=f'{qual.name} 将于 {qual.expiry_date} 过期',
                    severity='HIGH'
                ))
        
        return alerts
```

### 3. 供应商推荐 Agent

#### 功能
- 基于需求匹配供应商
- 多维度排序
- 智能推荐理由

#### 技术实现

```python
class SupplierRecommendationAgent:
    def __init__(self):
        self.profile_agent = SupplierProfileAgent()
        self.vector_db = VectorDatabase()
        self.reranker = RerankerModel()
    
    def recommend(self, requirement: Requirement) -> List[Recommendation]:
        """根据采购需求推荐供应商"""
        # 1. 基础筛选
        candidates = self.filter_by_basic_criteria(requirement)
        
        # 2. 向量检索（语义匹配）
        requirement_vector = self.embed_requirement(requirement)
        semantic_matches = self.vector_db.search(
            requirement_vector,
            filters={'category': requirement.category}
        )
        
        # 3. 合并候选
        all_candidates = self.merge_candidates(candidates, semantic_matches)
        
        # 4. 多维度评分
        scored = self.score_candidates(all_candidates, requirement)
        
        # 5. 重排序
        ranked = self.reranker.rerank(scored, requirement)
        
        # 6. 生成推荐理由
        for rec in ranked:
            rec.reason = self.generate_recommendation_reason(rec, requirement)
        
        return ranked[:10]
    
    def score_candidates(self, candidates: List[Supplier], 
                         requirement: Requirement) -> List[ScoredSupplier]:
        """多维度评分"""
        scored = []
        
        for supplier in candidates:
            profile = self.profile_agent.get_profile(supplier.id)
            
            score = SupplierScore(
                supplier=supplier,
                category_match=self.score_category_match(supplier, requirement),
                delivery_match=self.score_delivery(supplier, requirement),
                price_match=self.score_price(supplier, requirement),
                risk_score=profile.risk_score,
                quality_score=profile.quality_score,
                overall=0  # 待计算
            )
            
            # 加权计算总分
            score.overall = (
                score.category_match * 0.3 +
                score.delivery_match * 0.2 +
                score.price_match * 0.2 +
                (1 - score.risk_score) * 0.15 +
                score.quality_score * 0.15
            )
            
            scored.append(score)
        
        return scored
```

### 4. 报价分析 Agent

#### 功能
- 历史价格对比
- 市场价格分析
- 异常检测
- 价格合理性判断

#### 技术实现

```python
class PriceAnalysisAgent:
    def __init__(self):
        self.price_db = PriceDatabase()
        self.market_scraper = MarketPriceScraper()
        self.anomaly_detector = AnomalyDetector()
        self.llm = LLMClient()
    
    def analyze_price(self, quote: Quote) -> PriceAnalysis:
        """分析报价合理性"""
        analysis = PriceAnalysis(quote=quote)
        
        # 1. 历史价格对比
        historical = self.price_db.get_historical_prices(
            item=quote.item,
            supplier=quote.supplier,
            period='12m'
        )
        analysis.historical_comparison = self.compare_with_historical(
            quote.price, historical
        )
        
        # 2. 市场价格抓取
        market_prices = self.market_scraper.scrape(quote.item)
        analysis.market_comparison = self.compare_with_market(
            quote.price, market_prices
        )
        
        # 3. 异常检测
        analysis.is_anomaly = self.anomaly_detector.detect(
            quote.price, historical + market_prices
        )
        
        # 4. 价格合理性判断
        analysis.reasonableness = self.assess_reasonableness(analysis)
        
        # 5. 生成分析报告
        analysis.report = self.generate_report(analysis)
        
        return analysis
    
    def assess_reasonableness(self, analysis: PriceAnalysis) -> str:
        """判断价格合理性"""
        if analysis.is_anomaly:
            return 'UNREASONABLE'
        
        if analysis.historical_comparison.deviation > 0.3:
            return 'SUSPICIOUS'
        
        if analysis.market_comparison.deviation > 0.2:
            return 'ABOVE_MARKET'
        
        return 'REASONABLE'
```

### 5. 比价 Agent

#### 功能
- 多平台报价抓取
- 比价报告生成
- 最优推荐

#### 技术实现

```python
class ComparisonAgent:
    def __init__(self):
        self.scrapers = {
            'platform_a': PlatformAScraper(),
            'platform_b': PlatformBScraper(),
            'platform_c': PlatformCScraper()
        }
        self.price_analyzer = PriceAnalysisAgent()
        self.report_generator = ReportGenerator()
    
    def compare(self, item: str, quantity: int) -> ComparisonReport:
        """多平台比价"""
        # 并行抓取各平台报价
        with ThreadPoolExecutor() as executor:
            futures = {
                executor.submit(scraper.scrape, item, quantity): platform
                for platform, scraper in self.scrapers.items()
            }
            
            quotes = {}
            for future in as_completed(futures):
                platform = futures[future]
                quotes[platform] = future.result()
        
        # 分析每个报价
        analyses = {
            platform: self.price_analyzer.analyze_price(quote)
            for platform, quote in quotes.items()
        }
        
        # 排序推荐
        ranked = sorted(analyses.items(), 
                       key=lambda x: x[1].final_price)
        
        # 生成比价报告
        report = self.report_generator.generate(
            item=item,
            quantity=quantity,
            quotes=quotes,
            analyses=analyses,
            recommendation=ranked[0]
        )
        
        return report
```

### 6. 流程自动化 Agent

#### 功能
- PR 自动创建
- PO 自动生成
- 审批流程自动化
- 异常处理

#### 技术实现

```python
class ProcessAutomationAgent:
    def __init__(self):
        self.workflow_engine = WorkflowEngine()
        self.llm = LLMClient()
        self.notification_service = NotificationService()
    
    def auto_create_pr(self, request: PurchaseRequest) -> PR:
        """自动创建采购申请"""
        # 1. 验证请求
        validation = self.validate_request(request)
        if not validation.is_valid:
            raise InvalidRequestError(validation.errors)
        
        # 2. 智能填充缺失信息
        enriched = self.enrich_request(request)
        
        # 3. 创建 PR
        pr = PR.from_request(enriched)
        
        # 4. 启动审批流程
        self.workflow_engine.start(
            workflow='pr_approval',
            context={'pr': pr}
        )
        
        return pr
    
    def auto_create_po(self, pr: PR, selected_quote: Quote) -> PO:
        """自动创建采购订单"""
        # 1. 验证 PR 状态
        if pr.status != 'APPROVED':
            raise InvalidStateError('PR must be approved')
        
        # 2. 生成 PO
        po = PO(
            pr_id=pr.id,
            supplier=selected_quote.supplier,
            items=pr.items,
            prices=selected_quote.prices,
            delivery_date=selected_quote.delivery_date
        )
        
        # 3. 发送给供应商
        self.send_to_supplier(po)
        
        # 4. 启动跟踪
        self.workflow_engine.start(
            workflow='po_tracking',
            context={'po': po}
        )
        
        return po
```

### 7. 评标 Agent

#### 功能
- 自动评分
- 技术标评审
- 商务标评审
- 综合排名

#### 技术实现

```python
class EvaluationAgent:
    def __init__(self):
        self.technical_evaluator = TechnicalEvaluator()
        self.commercial_evaluator = CommercialEvaluator()
        self.llm = LLMClient()
    
    def evaluate_bids(self, tender: Tender, 
                      bids: List[Bid]) -> EvaluationResult:
        """评标"""
        results = []
        
        for bid in bids:
            # 技术标评审
            technical_score = self.technical_evaluator.evaluate(
                bid.technical_proposal,
                tender.technical_requirements
            )
            
            # 商务标评审
            commercial_score = self.commercial_evaluator.evaluate(
                bid.commercial_proposal,
                tender.budget
            )
            
            # 综合评分
            overall_score = (
                technical_score * tender.technical_weight +
                commercial_score * tender.commercial_weight
            )
            
            # 生成评语
            comment = self.generate_comment(
                bid, technical_score, commercial_score
            )
            
            results.append(BidEvaluation(
                bid=bid,
                technical_score=technical_score,
                commercial_score=commercial_score,
                overall_score=overall_score,
                comment=comment
            ))
        
        # 排序
        ranked = sorted(results, key=lambda x: x.overall_score, reverse=True)
        
        return EvaluationResult(
            tender=tender,
            evaluations=ranked,
            winner=ranked[0].bid if ranked else None
        )
```

## 三、技术栈选型

| 层级 | 技术选型 |
|------|----------|
| 前端 | React / Vue + Ant Design |
| 后端 | Python (FastAPI) / Go |
| AI 框架 | LangChain / AutoGen |
| LLM | GPT-4 / Claude / 通义千问 |
| 向量数据库 | Milvus / Pinecone / Weaviate |
| 知识图谱 | Neo4j / NebulaGraph |
| OCR | PaddleOCR / Tesseract |
| 工作流引擎 | Temporal / Airflow |
| 消息队列 | Kafka / RabbitMQ |
| 监控 | Prometheus + Grafana |

## 四、数据流设计

```
用户 → AI Agent → 向量数据库
         ↓
     知识图谱
         ↓
     外部数据源
         ↓
     业务数据库
         ↓
     用户
```

### 数据流说明

1. 用户发起采购请求
2. AI Agent 检索相似供应商
3. 查询供应商关系图谱
4. 抓取市场报价
5. 综合分析与推荐
6. 保存推荐结果
7. 返回推荐列表
8. 用户确认选择
9. 创建 PR/PO
10. 返回订单信息

## 五、部署架构

```
Kubernetes Cluster
├── Ingress
│   └── Nginx Ingress
├── AI Services
│   ├── Profile Agent
│   ├── Qualification Agent
│   ├── Recommendation Agent
│   ├── Price Analysis Agent
│   ├── Comparison Agent
│   ├── Process Agent
│   └── Evaluation Agent
├── Core Services
│   ├── API Gateway
│   ├── Supplier Service
│   ├── Purchase Service
│   └── Workflow Service
└── Data Layer
    ├── PostgreSQL
    ├── Milvus
    ├── Neo4j
    └── Redis
```

## 六、实施路线图

| 阶段 | 时间 | 交付物 |
|------|------|--------|
| Phase 1 | 1-2 月 | 基础架构搭建、供应商画像 Agent |
| Phase 2 | 3-4 月 | 资质审核 Agent、供应商推荐 Agent |
| Phase 3 | 5-6 月 | 报价分析 Agent、比价 Agent |
| Phase 4 | 7-8 月 | 流程自动化 Agent、评标 Agent |
| Phase 5 | 9-10 月 | 系统集成、测试与优化 |
| Phase 6 | 11-12 月 | 上线与运维 |

## 七、关键成功因素

1. **数据质量**：确保供应商数据的准确性和完整性
2. **模型调优**：根据业务反馈持续优化 AI 模型
3. **用户接受度**：提供可解释的推荐理由
4. **系统集成**：与现有 ERP/采购系统无缝对接
5. **安全合规**：确保数据安全和合规要求

## 八、预期收益

### 效率提升
- 供应商准入时间缩短 60%
- 报价分析时间缩短 80%
- 评标效率提升 70%



### 成本节约
- 采购成本降低 10-15%
- 管理成本降低 20%
- 风险损失降低 30%

### 质量改善
- 供应商质量提升 25%
- 合规性提升 40%
- 风险识别准确率提升 50%

---

*文档版本：v1.0*
*创建日期：2026-02-27*
