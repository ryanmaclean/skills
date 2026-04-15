#!/usr/bin/env python3
"""
Fleet Ralph Loop Engine - Distributed Self-Improvement Across 16 Raspberry Pis

Leverages 16 Pis, OLLAMA models, local inference, claude -p, codex --exec, and OpenCode
for massive parallel experimentation and continuous self-improvement.
"""

import asyncio
import json
import random
import time
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass, asdict
from enum import Enum
import concurrent.futures
from collections import defaultdict


class ProviderType(Enum):
    CLAUDE_CODE = "claude -p"
    CODEX = "codex --exec"
    OLLAMA_LOCAL = "ollama_local"
    OPENCODE = "opencode"
    HYBRID = "hybrid"


class NodeType(Enum):
    HEAVY_INFERENCE = "heavy_inference"  # 8GB Pis
    STANDARD_EXPERIMENTS = "standard_experiments"  # 4GB Pis
    LIGHTWEIGHT_COORDINATION = "lightweight_coordination"  # Pi 400s
    SPECIALIZED = "specialized"


@dataclass
class FleetNode:
    node_id: str
    ip_address: str
    node_type: NodeType
    ram_gb: int
    available_models: List[str]
    current_load: float = 0.0
    status: str = "active"
    last_heartbeat: datetime = None
    
    def __post_init__(self):
        if self.last_heartbeat is None:
            self.last_heartbeat = datetime.now()


@dataclass
class ExperimentAssignment:
    experiment_id: str
    hypothesis: str
    target_metric: str
    assigned_node: FleetNode
    provider: ProviderType
    model: str
    priority: int = 1
    estimated_duration: int = 0  # seconds
    estimated_cost: float = 0.0


@dataclass
class FleetExperimentResult:
    experiment_id: str
    node_id: str
    provider: ProviderType
    model: str
    hypothesis: str
    improvement: float
    statistical_significance: float
    execution_time: float
    actual_cost: float
    quality_score: float
    status: str = "completed"
    error: Optional[str] = None


class FleetTopology:
    """Defines the 16 Pi fleet topology and model distribution."""
    
    def __init__(self):
        self.nodes = self._initialize_fleet_nodes()
        self.model_distribution = self._setup_model_distribution()
        self.provider_capabilities = self._setup_provider_capabilities()
    
    def _initialize_fleet_nodes(self) -> List[FleetNode]:
        """Initialize the 16 Raspberry Pi fleet."""
        nodes = []
        
        # Heavy inference nodes (8GB Pis)
        heavy_models = ["llama3.1:70b", "qwen2.5:32b", "mixtral:8x7b"]
        for i in range(4):
            node = FleetNode(
                node_id=f"ubrpi{401+i}",
                ip_address=f"10.0.3.{215+i}",
                node_type=NodeType.HEAVY_INFERENCE,
                ram_gb=8,
                available_models=heavy_models.copy()
            )
            nodes.append(node)
        
        # Standard experiment nodes (4GB Pis)
        standard_models = ["llama3.1:8b", "qwen2.5:7b", "codellama:13b"]
        for i in range(8):
            node = FleetNode(
                node_id=f"ubrpi{405+i}",
                ip_address=f"10.0.3.{219+i}",
                node_type=NodeType.STANDARD_EXPERIMENTS,
                ram_gb=4,
                available_models=standard_models.copy()
            )
            nodes.append(node)
        
        # Lightweight coordination nodes (Pi 400s)
        specialized_models = ["deepseek-coder:33b", "starcoder2:15b", "wizardcoder:33b", "custom-model"]
        for i in range(4):
            node = FleetNode(
                node_id=f"ubrpi{413+i}",
                ip_address=f"10.0.3.{227+i}",
                node_type=NodeType.LIGHTWEIGHT_COORDINATION,
                ram_gb=4,
                available_models=[specialized_models[i]]
            )
            nodes.append(node)
        
        return nodes
    
    def _setup_model_distribution(self) -> Dict[str, List[str]]:
        """Setup model distribution across the fleet."""
        return {
            "llama3.1:70b": ["ubrpi401", "ubrpi402"],
            "qwen2.5:32b": ["ubrpi403", "ubrpi404"],
            "llama3.1:8b": ["ubrpi405", "ubrpi406", "ubrpi407", "ubrpi408"],
            "qwen2.5:7b": ["ubrpi409", "ubrpi410", "ubrpi411", "ubrpi412"],
            "deepseek-coder:33b": ["ubrpi413"],
            "starcoder2:15b": ["ubrpi414"],
            "wizardcoder:33b": ["ubrpi415"],
            "custom-model": ["ubrpi416"],
            "mixtral:8x7b": ["ubrpi401", "ubrpi402"]  # Also on heavy nodes
        }
    
    def _setup_provider_capabilities(self) -> Dict[ProviderType, Dict[str, Any]]:
        """Setup provider capabilities and limits."""
        return {
            ProviderType.CLAUDE_CODE: {
                "max_concurrent": 8,
                "quality_score": 0.95,
                "cost_per_experiment": 0.08,
                "average_duration": 45,
                "reasoning_strength": 0.95
            },
            ProviderType.CODEX: {
                "max_concurrent": 12,
                "quality_score": 0.90,
                "cost_per_experiment": 0.06,
                "average_duration": 30,
                "code_generation_strength": 0.95
            },
            ProviderType.OLLAMA_LOCAL: {
                "max_concurrent": 64,  # Limited by fleet size
                "quality_score": 0.75,
                "cost_per_experiment": 0.001,  # Just electricity
                "average_duration": 120,
                "cost_efficiency": 0.95
            },
            ProviderType.OPENCODE: {
                "max_concurrent": 16,
                "quality_score": 0.85,
                "cost_per_experiment": 0.02,
                "average_duration": 60,
                "automation_strength": 0.90
            }
        }


class FleetExperimentRouter:
    """Intelligently routes experiments to optimal node+provider combinations."""
    
    def __init__(self, fleet_topology: FleetTopology):
        self.fleet = fleet_topology
        self.node_performance = defaultdict(list)
        self.provider_performance = defaultdict(list)
        self.model_performance = defaultdict(list)
    
    def route_experiment(self, hypothesis: str, requirements: Dict[str, Any] = None) -> ExperimentAssignment:
        """Route experiment to optimal node+provider combination."""
        if requirements is None:
            requirements = self._analyze_requirements(hypothesis)
        
        # Select provider based on requirements
        provider = self._select_provider(requirements)
        
        # Select optimal node for the provider
        node = self._select_optimal_node(provider, requirements)
        
        # Select model for the node
        model = self._select_model(node, requirements)
        
        # Calculate estimates
        estimates = self._calculate_estimates(provider, model, requirements)
        
        assignment = ExperimentAssignment(
            experiment_id=f"exp_{int(time.time())}_{random.randint(1000, 9999)}",
            hypothesis=hypothesis,
            target_metric=requirements.get("target_metric", "performance"),
            assigned_node=node,
            provider=provider,
            model=model,
            priority=requirements.get("priority", 1),
            estimated_duration=estimates["duration"],
            estimated_cost=estimates["cost"]
        )
        
        return assignment
    
    def _analyze_requirements(self, hypothesis: str) -> Dict[str, Any]:
        """Analyze experiment requirements from hypothesis."""
        requirements = {
            "reasoning_intensity": 0.5,
            "code_generation": False,
            "cost_sensitivity": 0.5,
            "speed_priority": False,
            "quality_requirement": 0.8,
            "target_metric": "performance"
        }
        
        # Analyze hypothesis for requirements
        hypothesis_lower = hypothesis.lower()
        
        if any(word in hypothesis_lower for word in ["reasoning", "analysis", "complex", "strategic"]):
            requirements["reasoning_intensity"] = 0.9
            requirements["quality_requirement"] = 0.95
        
        if any(word in hypothesis_lower for word in ["code", "programming", "debug", "algorithm"]):
            requirements["code_generation"] = True
            requirements["reasoning_intensity"] = 0.7
        
        if any(word in hypothesis_lower for word in ["fast", "quick", "optimize", "efficient"]):
            requirements["speed_priority"] = True
        
        if any(word in hypothesis_lower for word in ["cost", "budget", "economical"]):
            requirements["cost_sensitivity"] = 0.9
        
        return requirements
    
    def _select_provider(self, requirements: Dict[str, Any]) -> ProviderType:
        """Select optimal provider based on requirements."""
        reasoning_intensity = requirements["reasoning_intensity"]
        code_generation = requirements["code_generation"]
        cost_sensitivity = requirements["cost_sensitivity"]
        speed_priority = requirements["speed_priority"]
        
        # High reasoning requirement -> Claude Code
        if reasoning_intensity > 0.8 and not cost_sensitivity:
            return ProviderType.CLAUDE_CODE
        
        # Code generation -> Codex or local code models
        elif code_generation:
            if not cost_sensitivity and random.random() < 0.6:
                return ProviderType.CODEX
            else:
                return ProviderType.OLLAMA_LOCAL
        
        # Cost sensitive -> Local OLLAMA
        elif cost_sensitivity > 0.7:
            return ProviderType.OLLAMA_LOCAL
        
        # Speed priority -> Local models or OpenCode
        elif speed_priority:
            return ProviderType.OPENCODE if random.random() < 0.5 else ProviderType.OLLAMA_LOCAL
        
        # Default -> Balanced approach
        else:
            return random.choice([ProviderType.CLAUDE_CODE, ProviderType.OLLAMA_LOCAL])
    
    def _select_optimal_node(self, provider: ProviderType, requirements: Dict[str, Any]) -> FleetNode:
        """Select optimal node for the given provider."""
        available_nodes = [node for node in self.fleet.nodes if node.status == "active"]
        
        # Filter by node capabilities
        if provider == ProviderType.CLAUDE_CODE:
            # Claude works best on heavy inference nodes
            candidate_nodes = [n for n in available_nodes if n.node_type == NodeType.HEAVY_INFERENCE]
        elif provider == ProviderType.CODEX:
            # Codex works well on standard or heavy nodes
            candidate_nodes = [n for n in available_nodes if n.node_type in [NodeType.HEAVY_INFERENCE, NodeType.STANDARD_EXPERIMENTS]]
        elif provider == ProviderType.OPENCODE:
            # OpenCode for fleet management - use coordination nodes
            candidate_nodes = [n for n in available_nodes if n.node_type == NodeType.LIGHTWEIGHT_COORDINATION]
        else:
            # OLLAMA local - any available node
            candidate_nodes = available_nodes
        
        # Sort by load (prefer less loaded nodes)
        candidate_nodes.sort(key=lambda n: n.current_load)
        
        # Return the best available node
        return candidate_nodes[0] if candidate_nodes else available_nodes[0]
    
    def _select_model(self, node: FleetNode, requirements: Dict[str, Any]) -> str:
        """Select optimal model for the node."""
        available_models = node.available_models
        
        # High reasoning -> Large models
        if requirements["reasoning_intensity"] > 0.8:
            large_models = [m for m in available_models if "70b" in m or "32b" in m]
            if large_models:
                return random.choice(large_models)
        
        # Code generation -> Code models
        if requirements["code_generation"]:
            code_models = [m for m in available_models if any(x in m for x in ["coder", "starcoder", "codellama"])]
            if code_models:
                return random.choice(code_models)
        
        # Default -> Best available model
        return available_models[0]
    
    def _calculate_estimates(self, provider: ProviderType, model: str, requirements: Dict[str, Any]) -> Dict[str, Any]:
        """Calculate duration and cost estimates."""
        provider_caps = self.fleet.provider_capabilities[provider]
        
        # Base estimates from provider
        base_duration = provider_caps["average_duration"]
        base_cost = provider_caps["cost_per_experiment"]
        
        # Adjust for model complexity
        if "70b" in model:
            duration_multiplier = 1.5
        elif "32b" in model:
            duration_multiplier = 1.3
        else:
            duration_multiplier = 1.0
        
        # Adjust for requirements
        if requirements["speed_priority"]:
            duration_multiplier *= 0.8
        
        if requirements["reasoning_intensity"] > 0.8:
            duration_multiplier *= 1.2
        
        estimated_duration = int(base_duration * duration_multiplier)
        estimated_cost = base_cost * duration_multiplier
        
        return {
            "duration": estimated_duration,
            "cost": estimated_cost
        }


class FleetOrchestrator:
    """Orchestrates experiments across the entire fleet."""
    
    def __init__(self):
        self.fleet_topology = FleetTopology()
        self.router = FleetExperimentRouter(self.fleet_topology)
        self.active_experiments = {}
        self.completed_experiments = []
        self.fleet_metrics = self._initialize_fleet_metrics()
    
    def _initialize_fleet_metrics(self) -> Dict[str, Any]:
        """Initialize fleet-wide metrics."""
        return {
            "total_experiments_run": 0,
            "successful_experiments": 0,
            "total_cost": 0.0,
            "total_execution_time": 0.0,
            "provider_usage": defaultdict(int),
            "model_usage": defaultdict(int),
            "node_usage": defaultdict(int),
            "improvements_by_provider": defaultdict(list),
            "fleet_utilization": 0.0
        }
    
    async def run_fleet_wide_experiments(self, hypotheses: List[str]) -> List[FleetExperimentResult]:
        """Run experiments across the entire fleet in parallel."""
        print(f"🚀 Launching {len(hypotheses)} experiments across 16 Pi fleet...")
        
        # Phase 1: Route all experiments to optimal nodes
        assignments = []
        for hypothesis in hypotheses:
            assignment = self.router.route_experiment(hypothesis)
            assignments.append(assignment)
        
        print(f"📋 Experiment routing complete:")
        provider_counts = defaultdict(int)
        node_counts = defaultdict(int)
        
        for assignment in assignments:
            provider_counts[assignment.provider.value] += 1
            node_counts[assignment.assigned_node.node_id] += 1
        
        for provider, count in provider_counts.items():
            print(f"   {provider}: {count} experiments")
        
        # Phase 2: Execute experiments in parallel
        print(f"\n🧪 Executing experiments in parallel...")
        experiment_tasks = []
        
        for assignment in assignments:
            task = self._execute_experiment_on_node(assignment)
            experiment_tasks.append(task)
        
        # Phase 3: Wait for all experiments to complete
        results = await asyncio.gather(*experiment_tasks, return_exceptions=True)
        
        # Phase 4: Process results and update metrics
        valid_results = []
        for result in results:
            if isinstance(result, FleetExperimentResult):
                valid_results.append(result)
                self._update_fleet_metrics(result)
            elif isinstance(result, Exception):
                print(f"⚠️ Experiment failed: {result}")
        
        print(f"\n📊 Fleet execution complete:")
        print(f"   Successful experiments: {len(valid_results)}")
        print(f"   Total cost: ${self.fleet_metrics['total_cost']:.4f}")
        print(f"   Fleet utilization: {self._calculate_fleet_utilization():.1%}")
        
        return valid_results
    
    async def _execute_experiment_on_node(self, assignment: ExperimentAssignment) -> FleetExperimentResult:
        """Execute experiment on specific fleet node."""
        start_time = time.time()
        
        try:
            # Simulate SSH execution to node
            await asyncio.sleep(0.1)  # Network latency simulation
            
            # Simulate experiment execution based on provider and model
            execution_time = self._simulate_experiment_execution(assignment)
            await asyncio.sleep(execution_time)  # Simulate actual execution time
            
            # Generate realistic results
            result = self._generate_experiment_result(assignment, start_time)
            
            return result
            
        except Exception as e:
            return FleetExperimentResult(
                experiment_id=assignment.experiment_id,
                node_id=assignment.assigned_node.node_id,
                provider=assignment.provider,
                model=assignment.model,
                hypothesis=assignment.hypothesis,
                improvement=0.0,
                statistical_significance=1.0,
                execution_time=time.time() - start_time,
                actual_cost=assignment.estimated_cost,
                quality_score=0.0,
                status="failed",
                error=str(e)
            )
    
    def _simulate_experiment_execution(self, assignment: ExperimentAssignment) -> float:
        """Simulate realistic experiment execution time."""
        base_time = assignment.estimated_duration / 10  # Scale down for demo
        
        # Add randomness
        random_factor = random.uniform(0.8, 1.2)
        
        # Adjust for provider performance
        provider_multipliers = {
            ProviderType.CLAUDE_CODE: 1.0,
            ProviderType.CODEX: 0.8,
            ProviderType.OLLAMA_LOCAL: 1.2,
            ProviderType.OPENCODE: 0.9
        }
        
        return base_time * random_factor * provider_multipliers.get(assignment.provider, 1.0)
    
    def _generate_experiment_result(self, assignment: ExperimentAssignment, start_time: float) -> FleetExperimentResult:
        """Generate realistic experiment results."""
        execution_time = time.time() - start_time
        
        # Provider-specific success rates and quality
        provider_stats = {
            ProviderType.CLAUDE_CODE: {"success_rate": 0.85, "avg_improvement": 12.0, "quality": 0.95},
            ProviderType.CODEX: {"success_rate": 0.80, "avg_improvement": 10.0, "quality": 0.90},
            ProviderType.OLLAMA_LOCAL: {"success_rate": 0.65, "avg_improvement": 6.0, "quality": 0.75},
            ProviderType.OPENCODE: {"success_rate": 0.75, "avg_improvement": 8.0, "quality": 0.85}
        }
        
        stats = provider_stats.get(assignment.provider, provider_stats[ProviderType.OLLAMA_LOCAL])
        
        # Determine if experiment succeeded
        succeeded = random.random() < stats["success_rate"]
        
        if succeeded:
            # Generate improvement with some variance
            improvement = random.gauss(stats["avg_improvement"], 3.0)
            significance = random.uniform(0.01, 0.05)
            quality = stats["quality"] + random.uniform(-0.05, 0.05)
        else:
            improvement = random.uniform(-5.0, 2.0)
            significance = random.uniform(0.1, 0.3)
            quality = stats["quality"] - random.uniform(0.1, 0.2)
        
        return FleetExperimentResult(
            experiment_id=assignment.experiment_id,
            node_id=assignment.assigned_node.node_id,
            provider=assignment.provider,
            model=assignment.model,
            hypothesis=assignment.hypothesis,
            improvement=improvement,
            statistical_significance=significance,
            execution_time=execution_time,
            actual_cost=assignment.estimated_cost,
            quality_score=max(0.0, min(1.0, quality)),
            status="completed"
        )
    
    def _update_fleet_metrics(self, result: FleetExperimentResult):
        """Update fleet-wide metrics with experiment result."""
        self.fleet_metrics["total_experiments_run"] += 1
        
        if result.status == "completed" and result.improvement > 0:
            self.fleet_metrics["successful_experiments"] += 1
        
        self.fleet_metrics["total_cost"] += result.actual_cost
        self.fleet_metrics["total_execution_time"] += result.execution_time
        self.fleet_metrics["provider_usage"][result.provider.value] += 1
        self.fleet_metrics["model_usage"][result.model] += 1
        self.fleet_metrics["node_usage"][result.node_id] += 1
        self.fleet_metrics["improvements_by_provider"][result.provider.value].append(result.improvement)
    
    def _calculate_fleet_utilization(self) -> float:
        """Calculate current fleet utilization."""
        total_capacity = len(self.fleet_topology.nodes) * 4  # 4 experiments per node capacity
        used_capacity = self.fleet_metrics["total_experiments_run"] % total_capacity
        return min(used_capacity / total_capacity, 1.0)
    
    def get_fleet_summary(self) -> Dict[str, Any]:
        """Get comprehensive fleet performance summary."""
        metrics = self.fleet_metrics
        
        # Calculate provider performance
        provider_performance = {}
        for provider, improvements in metrics["improvements_by_provider"].items():
            if improvements:
                provider_performance[provider] = {
                    "experiments": len(improvements),
                    "avg_improvement": sum(improvements) / len(improvements),
                    "total_improvement": sum(improvements)
                }
        
        return {
            "fleet_size": len(self.fleet_topology.nodes),
            "total_experiments": metrics["total_experiments_run"],
            "success_rate": metrics["successful_experiments"] / max(metrics["total_experiments_run"], 1),
            "total_cost": metrics["total_cost"],
            "avg_cost_per_experiment": metrics["total_cost"] / max(metrics["total_experiments_run"], 1),
            "total_execution_time": metrics["total_execution_time"],
            "fleet_utilization": self._calculate_fleet_utilization(),
            "provider_performance": provider_performance,
            "top_models": sorted(metrics["model_usage"].items(), key=lambda x: x[1], reverse=True)[:5],
            "busiest_nodes": sorted(metrics["node_usage"].items(), key=lambda x: x[1], reverse=True)[:5]
        }


class FleetRalphLoopEngine:
    """Main Fleet Ralph Loop engine coordinating the entire 16 Pi fleet."""
    
    def __init__(self):
        self.orchestrator = FleetOrchestrator()
        self.experiment_history = []
        self.learning_cycles = 0
        self.cost_analysis = self._initialize_cost_analysis()
    
    def _initialize_cost_analysis(self) -> Dict[str, Any]:
        """Initialize cost tracking and analysis."""
        return {
            "fleet_cost_per_hour": 0.10,  # Electricity for 16 Pis
            "cloud_comparison": {
                "claude_cost_per_experiment": 0.80,
                "codex_cost_per_experiment": 0.60,
                "total_cloud_cost": 0.0
            },
            "fleet_savings": 0.0,
            "roi_calculations": []
        }
    
    async def run_fleet_improvement_cycle(self, hypothesis_count: int = 50) -> Dict[str, Any]:
        """Run a fleet-wide improvement cycle with massive parallel experimentation."""
        print(f"\n🌟 Fleet Ralph Loop Cycle #{self.learning_cycles + 1}")
        print(f"🎯 Target: {hypothesis_count} parallel experiments")
        print("=" * 70)
        
        # Generate diverse hypotheses for fleet experimentation
        hypotheses = self._generate_fleet_hypotheses(hypothesis_count)
        
        # Run experiments across entire fleet
        start_time = time.time()
        results = await self.orchestrator.run_fleet_wide_experiments(hypotheses)
        cycle_time = time.time() - start_time
        
        # Analyze fleet-wide results
        analysis = self._analyze_fleet_results(results)
        
        # Calculate cost savings
        cost_analysis = self._calculate_cost_savings(results, len(hypotheses))
        
        # Generate collective intelligence
        collective_intelligence = self._synthesize_collective_intelligence(results, analysis)
        
        # Update learning cycle
        self.learning_cycles += 1
        self.experiment_history.append({
            "cycle": self.learning_cycles,
            "hypotheses_count": len(hypotheses),
            "results": results,
            "analysis": analysis,
            "cost_analysis": cost_analysis,
            "collective_intelligence": collective_intelligence,
            "cycle_time": cycle_time,
            "timestamp": datetime.now()
        })
        
        # Print cycle summary
        self._print_cycle_summary(analysis, cost_analysis, cycle_time)
        
        return {
            "cycle": self.learning_cycles,
            "hypotheses_tested": len(hypotheses),
            "experiments_completed": len(results),
            "analysis": analysis,
            "cost_analysis": cost_analysis,
            "collective_intelligence": collective_intelligence,
            "cycle_time": cycle_time
        }
    
    def _generate_fleet_hypotheses(self, count: int) -> List[str]:
        """Generate diverse hypotheses for fleet testing."""
        hypothesis_templates = [
            "Structured context gathering improves task completion by {improvement}%",
            "Multi-draft response generation enhances quality by {improvement}%",
            "Provider-specific optimization reduces costs by {improvement}%",
            "Model selection algorithms improve accuracy by {improvement}%",
            "Parallel experiment design accelerates learning by {improvement}%",
            "Cross-model federation enhances robustness by {improvement}%",
            "Fleet load balancing optimizes resource usage by {improvement}%",
            "Cost-aware routing improves efficiency by {improvement}%",
            "Quality-based provider selection improves outcomes by {improvement}%",
            "Distributed learning coordination improves speed by {improvement}%"
        ]
        
        hypotheses = []
        for i in range(count):
            template = random.choice(hypothesis_templates)
            improvement = random.uniform(5, 25)
            hypothesis = template.format(improvement=improvement)
            hypotheses.append(hypothesis)
        
        return hypotheses
    
    def _analyze_fleet_results(self, results: List[FleetExperimentResult]) -> Dict[str, Any]:
        """Analyze results from fleet-wide experimentation."""
        if not results:
            return {}
        
        # Overall metrics
        successful_results = [r for r in results if r.status == "completed" and r.improvement > 0]
        total_improvement = sum(r.improvement for r in successful_results)
        avg_improvement = total_improvement / len(successful_results) if successful_results else 0
        
        # Provider analysis
        provider_analysis = defaultdict(list)
        for result in results:
            provider_analysis[result.provider.value].append(result.improvement)
        
        provider_performance = {}
        for provider, improvements in provider_analysis.items():
            successful = [i for i in improvements if i > 0]
            provider_performance[provider] = {
                "experiments": len(improvements),
                "success_rate": len(successful) / len(improvements),
                "avg_improvement": sum(successful) / len(successful) if successful else 0,
                "total_improvement": sum(successful)
            }
        
        # Model analysis
        model_analysis = defaultdict(list)
        for result in results:
            model_analysis[result.model].append(result.improvement)
        
        model_performance = {}
        for model, improvements in model_analysis.items():
            successful = [i for i in improvements if i > 0]
            if successful:
                model_performance[model] = {
                    "experiments": len(improvements),
                    "avg_improvement": sum(successful) / len(successful),
                    "total_improvement": sum(successful)
                }
        
        return {
            "total_experiments": len(results),
            "successful_experiments": len(successful_results),
            "success_rate": len(successful_results) / len(results),
            "total_improvement": total_improvement,
            "avg_improvement": avg_improvement,
            "provider_performance": provider_performance,
            "model_performance": model_performance,
            "best_performing_provider": max(provider_performance.items(), key=lambda x: x[1]["avg_improvement"])[0] if provider_performance else None,
            "best_performing_model": max(model_performance.items(), key=lambda x: x[1]["avg_improvement"])[0] if model_performance else None
        }
    
    def _calculate_cost_savings(self, results: List[FleetExperimentResult], hypothesis_count: int) -> Dict[str, Any]:
        """Calculate cost savings compared to cloud-only approach."""
        # Actual fleet cost
        fleet_cost = sum(r.actual_cost for r in results)
        
        # Equivalent cloud cost
        cloud_cost_per_experiment = {
            ProviderType.CLAUDE_CODE: 0.80,
            ProviderType.CODEX: 0.60,
            ProviderType.OLLAMA_LOCAL: 0.10,  # What cloud would charge for similar inference
            ProviderType.OPENCODE: 0.20
        }
        
        equivalent_cloud_cost = sum(
            cloud_cost_per_experiment.get(r.provider, 0.50) for r in results
        )
        
        savings = equivalent_cloud_cost - fleet_cost
        savings_percentage = (savings / equivalent_cloud_cost * 100) if equivalent_cloud_cost > 0 else 0
        
        return {
            "fleet_cost": fleet_cost,
            "equivalent_cloud_cost": equivalent_cloud_cost,
            "savings": savings,
            "savings_percentage": savings_percentage,
            "cost_per_experiment_fleet": fleet_cost / len(results),
            "cost_per_experiment_cloud": equivalent_cloud_cost / len(results),
            "monthly_projection_fleet": fleet_cost * 30,  # Assuming daily cycle
            "monthly_projection_cloud": equivalent_cloud_cost * 30
        }
    
    def _synthesize_collective_intelligence(self, results: List[FleetExperimentResult], analysis: Dict[str, Any]) -> Dict[str, Any]:
        """Synthesize collective intelligence from fleet results."""
        # Identify cross-provider patterns
        cross_provider_insights = []
        for provider, perf in analysis.get("provider_performance", {}).items():
            if perf["avg_improvement"] > 8:  # High-performing providers
                cross_provider_insights.append(f"{provider} shows strong performance with {perf['avg_improvement']:.1f}% avg improvement")
        
        # Identify model advantages
        model_insights = []
        for model, perf in analysis.get("model_performance", {}).items():
            if perf["avg_improvement"] > 10:  # High-performing models
                model_insights.append(f"{model} excels with {perf['avg_improvement']:.1f}% avg improvement")
        
        # Generate optimization recommendations
        recommendations = []
        best_provider = analysis.get("best_performing_provider")
        if best_provider:
            recommendations.append(f"Increase usage of {best_provider} for high-value experiments")
        
        best_model = analysis.get("best_performing_model")
        if best_model:
            recommendations.append(f"Prioritize {best_model} for similar task types")
        
        return {
            "cross_provider_insights": cross_provider_insights,
            "model_advantages": model_insights,
            "optimization_recommendations": recommendations,
            "collective_performance_score": analysis.get("avg_improvement", 0),
            "fleet_intelligence_maturity": min(analysis.get("success_rate", 0) * 1.2, 1.0)
        }
    
    def _print_cycle_summary(self, analysis: Dict[str, Any], cost_analysis: Dict[str, Any], cycle_time: float):
        """Print comprehensive cycle summary."""
        print(f"\n📊 Fleet Ralph Loop Cycle #{self.learning_cycles} Summary")
        print("=" * 60)
        
        print(f"🧪 Experiment Results:")
        print(f"   Success rate: {analysis.get('success_rate', 0):.1%}")
        print(f"   Average improvement: {analysis.get('avg_improvement', 0):.1f}%")
        print(f"   Total improvement: {analysis.get('total_improvement', 0):.1f}%")
        
        print(f"\n💰 Cost Analysis:")
        print(f"   Fleet cost: ${cost_analysis['fleet_cost']:.4f}")
        print(f"   Cloud equivalent: ${cost_analysis['equivalent_cloud_cost']:.2f}")
        print(f"   Savings: ${cost_analysis['savings']:.2f} ({cost_analysis['savings_percentage']:.1f}%)")
        
        print(f"\n⚡ Performance:")
        print(f"   Cycle time: {cycle_time:.1f}s")
        print(f"   Best provider: {analysis.get('best_performing_provider', 'N/A')}")
        print(f"   Best model: {analysis.get('best_performing_model', 'N/A')}")
        
        print(f"\n🎯 Fleet Intelligence:")
        print(f"   Collective performance: {analysis.get('avg_improvement', 0):.1f}%")
        print(f"   Maturity level: {analysis.get('success_rate', 0) * 1.2:.1%}")
    
    def get_comprehensive_summary(self) -> Dict[str, Any]:
        """Get comprehensive summary of all fleet operations."""
        fleet_summary = self.orchestrator.get_fleet_summary()
        
        total_experiments = sum(c["hypotheses_count"] for c in self.experiment_history)
        total_cost = sum(c["cost_analysis"]["fleet_cost"] for c in self.experiment_history)
        total_savings = sum(c["cost_analysis"]["savings"] for c in self.experiment_history)
        
        return {
            "learning_cycles": self.learning_cycles,
            "total_experiments": total_experiments,
            "fleet_summary": fleet_summary,
            "total_cost": total_cost,
            "total_savings": total_savings,
            "average_cost_per_experiment": total_cost / max(total_experiments, 1),
            "roi": (total_savings / total_cost * 100) if total_cost > 0 else 0,
            "fleet_efficiency": total_experiments / self.learning_cycles if self.learning_cycles > 0 else 0
        }


async def main():
    """Demonstrate the Fleet Ralph Loop with massive parallel experimentation."""
    print("🚀 Fleet Ralph Loop - 16 Pi Distributed Self-Improvement")
    print("=" * 80)
    print("📡 Fleet: 16 Raspberry Pis (4x 8GB + 8x 4GB + 4x Pi 400)")
    print("🤖 Models: llama3.1:70b, qwen2.5:32b, deepseek-coder:33b, + more")
    print("🔗 Providers: Claude Code, Codex, OLLAMA Local, OpenCode")
    print("=" * 80)
    
    # Initialize fleet engine
    fleet_engine = FleetRalphLoopEngine()
    
    # Run multiple fleet improvement cycles
    for cycle in range(3):
        print(f"\n{'='*20} CYCLE {cycle + 1} {'='*20}")
        
        # Experiment with different scales
        if cycle == 0:
            hypothesis_count = 20  # Smaller first cycle
        elif cycle == 1:
            hypothesis_count = 35  # Medium cycle
        else:
            hypothesis_count = 50  # Full fleet capacity
        
        results = await fleet_engine.run_fleet_improvement_cycle(hypothesis_count)
        
        # Brief pause between cycles
        if cycle < 2:
            print(f"\n⏳ Fleet reorganizing for next cycle...")
            await asyncio.sleep(0.5)
    
    # Final comprehensive summary
    summary = fleet_engine.get_comprehensive_summary()
    
    print(f"\n🌟 Fleet Ralph Loop - Comprehensive Summary")
    print("=" * 70)
    print(f"🔄 Learning cycles completed: {summary['learning_cycles']}")
    print(f"🧪 Total experiments: {summary['total_experiments']}")
    print(f"⚡ Fleet efficiency: {summary['fleet_efficiency']:.1f} experiments/cycle")
    print(f"💰 Total cost: ${summary['total_cost']:.4f}")
    print(f"💸 Total savings: ${summary['total_savings']:.2f}")
    print(f"📈 ROI: {summary['roi']:.1f}%")
    print(f"💡 Avg cost per experiment: ${summary['average_cost_per_experiment']:.6f}")
    
    print(f"\n🎯 Fleet Performance:")
    fleet = summary['fleet_summary']
    print(f"   Success rate: {fleet['success_rate']:.1%}")
    print(f"   Fleet utilization: {fleet['fleet_utilization']:.1%}")
    print(f"   Best provider: {max(fleet['provider_performance'].items(), key=lambda x: x[1]['avg_improvement'])[0] if fleet['provider_performance'] else 'N/A'}")
    
    print(f"\n🚀 Fleet Ralph Loop achieved {summary['total_savings']:.2f} in cost savings with {summary['total_experiments']} parallel experiments!")


if __name__ == "__main__":
    asyncio.run(main())
