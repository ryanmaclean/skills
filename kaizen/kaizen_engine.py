#!/usr/bin/env python3
"""
Kaizen Engine - Continuous Self-Improvement System

Implements the Kaizen philosophy of continuous improvement through
iterative experimentation, evaluation, and optimization.
"""

import json
import time
import random
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass, asdict
from enum import Enum


class ExperimentStatus(Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    ADOPTED = "adopted"
    REJECTED = "rejected"


@dataclass
class PerformanceMetrics:
    task_completion_rate: float = 0.0
    user_satisfaction: float = 0.0
    response_relevance: float = 0.0
    error_rate: float = 0.0
    efficiency_score: float = 0.0
    clarification_rate: float = 0.0
    response_time: float = 0.0
    token_efficiency: float = 0.0


@dataclass
class Experiment:
    name: str
    hypothesis: str
    target_metric: str
    expected_improvement: float
    changes: List[str]
    sample_size: int
    duration_hours: int
    status: ExperimentStatus = ExperimentStatus.PENDING
    results: Optional[Dict[str, Any]] = None
    statistical_significance: float = 0.0
    actual_improvement: float = 0.0


class MetricsCollector:
    """Collects and tracks performance metrics for kaizen evaluation."""
    
    def __init__(self):
        self.interaction_log = []
        self.metrics_history = []
        self.current_session = {
            "start_time": datetime.now(),
            "interactions": [],
            "metrics": PerformanceMetrics()
        }
    
    def log_interaction(self, user_input: str, my_response: str, 
                       user_feedback: Optional[int] = None,
                       tools_used: List[str] = None,
                       response_time: float = 0.0,
                       tokens_used: int = 0):
        """Log an interaction for metric calculation."""
        
        interaction = {
            "timestamp": datetime.now(),
            "user_input": user_input,
            "my_response": my_response,
            "user_feedback": user_feedback,
            "tools_used": tools_used or [],
            "response_time": response_time,
            "tokens_used": tokens_used,
            "required_clarification": self._requires_clarification(user_input, my_response),
            "task_completed": self._task_completed(user_input, my_response),
            "response_relevant": self._calculate_relevance(user_input, my_response)
        }
        
        self.interaction_log.append(interaction)
        self.current_session["interactions"].append(interaction)
        
        # Update session metrics
        self._update_session_metrics()
    
    def _requires_clarification(self, user_input: str, my_response: str) -> bool:
        """Determine if I had to ask for clarification."""
        clarification_indicators = [
            "Could you clarify", "What do you mean", "Can you provide more details",
            "I need more information", "Please specify", "Could you elaborate"
        ]
        
        return any(indicator in my_response for indicator in clarification_indicators)
    
    def _task_completed(self, user_input: str, my_response: str) -> bool:
        """Estimate if the task was completed successfully."""
        completion_indicators = [
            "Here's the solution", "I've completed", "The task is done",
            "Successfully created", "Finished", "Complete", "Resolved"
        ]
        
        return any(indicator in my_response for indicator in completion_indicators)
    
    def _calculate_relevance(self, user_input: str, my_response: str) -> float:
        """Calculate semantic relevance (simplified version)."""
        user_terms = set(user_input.lower().split())
        response_terms = set(my_response.lower().split())
        
        intersection = len(user_terms.intersection(response_terms))
        union = len(user_terms.union(response_terms))
        
        return intersection / union if union > 0 else 0.0
    
    def _update_session_metrics(self):
        """Update current session metrics based on interactions."""
        interactions = self.current_session["interactions"]
        
        if not interactions:
            return
        
        completed = sum(1 for i in interactions if i["task_completed"])
        self.current_session["metrics"].task_completion_rate = completed / len(interactions)
        
        clarifications = sum(1 for i in interactions if i["required_clarification"])
        self.current_session["metrics"].clarification_rate = clarifications / len(interactions)
        
        feedback_scores = [i["user_feedback"] for i in interactions if i["user_feedback"]]
        if feedback_scores:
            self.current_session["metrics"].user_satisfaction = sum(feedback_scores) / len(feedback_scores)
        
        relevance_scores = [i["response_relevant"] for i in interactions]
        self.current_session["metrics"].response_relevance = sum(relevance_scores) / len(relevance_scores)
        
        response_times = [i["response_time"] for i in interactions if i["response_time"] > 0]
        if response_times:
            self.current_session["metrics"].response_time = sum(response_times) / len(response_times)
        
        total_tokens = sum(i["tokens_used"] for i in interactions if i["tokens_used"] > 0)
        completed_tasks = sum(1 for i in interactions if i["task_completed"] and i["tokens_used"] > 0)
        if total_tokens > 0 and completed_tasks > 0:
            self.current_session["metrics"].token_efficiency = completed_tasks / total_tokens
    
    def get_current_metrics(self) -> PerformanceMetrics:
        """Get current performance metrics."""
        return self.current_session["metrics"]
    
    def get_baseline_metrics(self, days: int = 7) -> PerformanceMetrics:
        """Get baseline metrics from recent history."""
        cutoff_date = datetime.now() - timedelta(days=days)
        recent_interactions = [
            i for i in self.interaction_log 
            if i["timestamp"] > cutoff_date
        ]
        
        if not recent_interactions:
            return PerformanceMetrics()
        
        metrics = PerformanceMetrics()
        
        completed = sum(1 for i in recent_interactions if i["task_completed"])
        metrics.task_completion_rate = completed / len(recent_interactions)
        
        clarifications = sum(1 for i in recent_interactions if i["required_clarification"])
        metrics.clarification_rate = clarifications / len(recent_interactions)
        
        feedback_scores = [i["user_feedback"] for i in recent_interactions if i["user_feedback"]]
        if feedback_scores:
            metrics.user_satisfaction = sum(feedback_scores) / len(feedback_scores)
        
        relevance_scores = [i["response_relevant"] for i in recent_interactions]
        metrics.response_relevance = sum(relevance_scores) / len(relevance_scores)
        
        return metrics


class ExperimentManager:
    """Manages kaizen experiments."""
    
    def __init__(self):
        self.experiments = []
        self.active_experiment = None
        self.experiment_results = []
    
    def generate_hypotheses(self, metrics: PerformanceMetrics) -> List[str]:
        """Generate kaizen hypotheses based on performance gaps."""
        hypotheses = []
        
        if metrics.task_completion_rate < 0.9:
            hypotheses.append(
                "Adding structured context gathering will improve task completion rate by 5%"
            )
        
        if metrics.clarification_rate > 0.2:
            hypotheses.append(
                "Implementing context validation will reduce clarification requests by 30%"
            )
        
        if metrics.user_satisfaction < 4.5:
            hypotheses.append(
                "Using multi-draft approach with self-critique will improve user satisfaction by 0.3 points"
            )
        
        if metrics.response_relevance < 0.85:
            hypotheses.append(
                "Adding intent analysis before response generation will improve relevance by 10%"
            )
        
        return hypotheses
    
    def create_experiment(self, hypothesis: str, metrics: PerformanceMetrics) -> Experiment:
        """Create an experiment from a hypothesis."""
        experiment = Experiment(
            name=f"kaizen_exp_{datetime.now().strftime('%Y%m%d_%H%M%S')}",
            hypothesis=hypothesis,
            target_metric=self._extract_target_metric(hypothesis),
            expected_improvement=self._extract_improvement(hypothesis),
            changes=self._design_changes(hypothesis),
            sample_size=20,
            duration_hours=24
        )
        
        return experiment
    
    def _extract_target_metric(self, hypothesis: str) -> str:
        """Extract the target metric from a hypothesis."""
        if "completion rate" in hypothesis:
            return "task_completion_rate"
        elif "clarification" in hypothesis:
            return "clarification_rate"
        elif "satisfaction" in hypothesis:
            return "user_satisfaction"
        elif "relevance" in hypothesis:
            return "response_relevance"
        else:
            return "efficiency_score"
    
    def _extract_improvement(self, hypothesis: str) -> float:
        """Extract expected improvement percentage from hypothesis."""
        import re
        
        match = re.search(r'(\d+)%', hypothesis)
        if match:
            return float(match.group(1)) / 100
        
        match = re.search(r'(\d+\.\d+) points?', hypothesis)
        if match:
            return float(match.group(1)) / 5.0
        
        return 0.05
    
    def _design_changes(self, hypothesis: str) -> List[str]:
        """Design specific changes for an experiment."""
        if "context gathering" in hypothesis:
            return [
                "Add mandatory context clarification step",
                "Use structured context template",
                "Validate context completeness before proceeding"
            ]
        elif "context validation" in hypothesis:
            return [
                "Pre-validate user intent",
                "Check for missing information",
                "Request specific details proactively"
            ]
        elif "multi-draft" in hypothesis:
            return [
                "Generate 3 response drafts",
                "Self-critique each draft",
                "Select and refine best draft"
            ]
        elif "intent analysis" in hypothesis:
            return [
                "Extract key intent components",
                "Map intent to response structure",
                "Validate alignment before generation"
            ]
        else:
            return ["Generic kaizen improvement change"]
    
    def run_experiment(self, experiment: Experiment, 
                      baseline_metrics: PerformanceMetrics) -> Dict[str, Any]:
        """Simulate running an experiment and return results."""
        time.sleep(0.1)
        
        improvement_chance = random.uniform(0.3, 0.7)
        
        if improvement_chance > 0.5:
            actual_improvement = experiment.expected_improvement * random.uniform(0.5, 1.2)
            statistical_significance = random.uniform(0.02, 0.05)
        else:
            actual_improvement = experiment.expected_improvement * random.uniform(-0.3, 0.2)
            statistical_significance = random.uniform(0.1, 0.3)
        
        results = {
            "baseline": asdict(baseline_metrics),
            "experiment": {
                "target_metric": experiment.target_metric,
                "actual_improvement": actual_improvement,
                "statistical_significance": statistical_significance,
                "sample_size": experiment.sample_size,
                "duration_hours": experiment.duration_hours
            },
            "recommendation": "adopt" if actual_improvement > 0 and statistical_significance < 0.05 else "reject"
        }
        
        experiment.results = results
        experiment.actual_improvement = actual_improvement
        experiment.statistical_significance = statistical_significance
        experiment.status = ExperimentStatus.COMPLETED
        
        return results


class KaizenEngine:
    """Main Kaizen continuous improvement engine."""
    
    def __init__(self):
        self.metrics_collector = MetricsCollector()
        self.experiment_manager = ExperimentManager()
        self.learning_history = []
        self.current_approach = "standard"
        self.kaizen_cycles = 0
        
    def process_interaction(self, user_input: str, my_response: str,
                           user_feedback: Optional[int] = None) -> str:
        """Process an interaction and collect metrics."""
        start_time = time.time()
        
        self.metrics_collector.log_interaction(
            user_input=user_input,
            my_response=my_response,
            user_feedback=user_feedback,
            response_time=time.time() - start_time,
            tokens_used=len(my_response.split()) * 2
        )
        
        if self._should_run_kaizen_cycle():
            self._run_kaizen_cycle()
        
        return my_response
    
    def _should_run_kaizen_cycle(self) -> bool:
        """Determine if it's time to run a kaizen cycle."""
        interactions = self.metrics_collector.current_session["interactions"]
        session_duration = datetime.now() - self.metrics_collector.current_session["start_time"]
        
        return (len(interactions) >= 20 or session_duration.total_seconds() > 3600)
    
    def _run_kaizen_cycle(self):
        """Run a complete kaizen improvement cycle."""
        print("\n🔄 Running Kaizen Improvement Cycle...")
        
        current_metrics = self.metrics_collector.get_current_metrics()
        baseline_metrics = self.metrics_collector.get_baseline_metrics()
        
        print(f"📊 Current Metrics: {current_metrics.task_completion_rate:.1%} completion rate")
        
        hypotheses = self.experiment_manager.generate_hypotheses(current_metrics)
        
        if not hypotheses:
            print("✅ Performance is optimal - no kaizen improvements needed")
            return
        
        print(f"💡 Generated {len(hypotheses)} kaizen hypotheses")
        
        hypothesis = hypotheses[0]
        experiment = self.experiment_manager.create_experiment(hypothesis, current_metrics)
        
        print(f"🧪 Testing: {hypothesis}")
        
        results = self.experiment_manager.run_experiment(experiment, baseline_metrics)
        
        improvement = results["experiment"]["actual_improvement"]
        significance = results["experiment"]["statistical_significance"]
        recommendation = results["recommendation"]
        
        print(f"📈 Result: {improvement:+.1%} improvement (p={significance:.3f})")
        
        if recommendation == "adopt":
            self._adopt_improvement(experiment)
            print(f"✅ Adopted kaizen improvement: {experiment.name}")
        else:
            print(f"❌ Rejected improvement: {experiment.name}")
        
        self.learning_history.append({
            "cycle": self.kaizen_cycles,
            "timestamp": datetime.now(),
            "hypothesis": hypothesis,
            "results": results,
            "adopted": recommendation == "adopt"
        })
        
        self.kaizen_cycles += 1
        
        print(f"🎯 Total kaizen cycles: {self.kaizen_cycles}")
    
    def _adopt_improvement(self, experiment: Experiment):
        """Adopt a successful kaizen improvement."""
        experiment.status = ExperimentStatus.ADOPTED
        self.current_approach = f"kaizen_enhanced_v{self.kaizen_cycles + 1}"
    
    def get_kaizen_summary(self) -> Dict[str, Any]:
        """Get a summary of kaizen improvement efforts."""
        adopted_experiments = [exp for exp in self.experiment_manager.experiments 
                             if exp.status == ExperimentStatus.ADOPTED]
        
        total_improvement = sum(exp.actual_improvement for exp in adopted_experiments)
        
        return {
            "cycles_completed": self.kaizen_cycles,
            "experiments_run": len(self.experiment_manager.experiments),
            "improvements_adopted": len(adopted_experiments),
            "total_improvement": total_improvement,
            "current_approach": self.current_approach,
            "success_rate": len(adopted_experiments) / len(self.experiment_manager.experiments) if self.experiment_manager.experiments else 0
        }


def main():
    """Demonstrate the Kaizen engine in action."""
    print("🚀 Kaizen Engine - Continuous Self-Improvement")
    print("=" * 50)
    
    kaizen = KaizenEngine()
    
    interactions = [
        ("Help me write a Python script to process CSV files", "Here's a Python script that processes CSV files using pandas..."),
        ("What's the difference between list and tuple in Python?", "Lists are mutable while tuples are immutable..."),
        ("How do I optimize database queries?", "To optimize database queries, consider indexing..."),
        ("Explain machine learning to me", "Machine learning is a subset of AI that enables computers..."),
        ("Help me debug this error", "Looking at the error, it seems like there's a syntax issue...")
    ]
    
    print("📝 Processing sample interactions...")
    
    for i, (user_input, my_response) in enumerate(interactions):
        user_feedback = random.randint(3, 5)
        
        kaizen.process_interaction(user_input, my_response, user_feedback)
        print(f"  Interaction {i+1}: Processed (feedback: {user_feedback}/5)")
    
    kaizen._run_kaizen_cycle()
    
    summary = kaizen.get_kaizen_summary()
    
    print(f"\n📊 Kaizen Summary:")
    print(f"  Cycles completed: {summary['cycles_completed']}")
    print(f"  Experiments run: {summary['experiments_run']}")
    print(f"  Improvements adopted: {summary['improvements_adopted']}")
    print(f"  Total improvement: {summary['total_improvement']:+.1%}")
    print(f"  Success rate: {summary['success_rate']:.1%}")
    print(f"  Current approach: {summary['current_approach']}")
    
    print(f"\n🎯 Kaizen is continuously improving!")


if __name__ == "__main__":
    main()
