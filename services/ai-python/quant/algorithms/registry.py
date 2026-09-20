from .daily_observe import DailyObserveAlgorithm
from .deep_rebound import DeepReboundAlgorithm
from .forward_train import ForwardTrainAlgorithm
from .ma5_align import Ma5AlignAlgorithm
from .year_high import YearHighAlgorithm

_ALGOS = [
    YearHighAlgorithm(),
    DeepReboundAlgorithm(),
    ForwardTrainAlgorithm(),
    DailyObserveAlgorithm(),
    Ma5AlignAlgorithm(),
]

REGISTRY = {a.code: a for a in _ALGOS}

DEFAULT_PROFILE = {
    "code": "screening_default",
    "mode": "ensemble",
    "algorithms": [
        {"code": "year_high", "weight": 1.0, "baseScore": 46},
        {"code": "deep_rebound", "weight": 1.0, "baseScore": 40},
        {"code": "forward_train", "weight": 1.0, "baseScore": 36},
        {"code": "daily_observe", "weight": 1.0, "baseScore": 30},
        {"code": "ma5_align", "weight": 1.0, "baseScore": 24},
    ],
    "fusion": {"setLogic": "weighted_rank", "limit": 30, "crashDropPct": -7},
}


def list_algorithms() -> list[dict]:
    return [
        {
            "code": a.code,
            "name": a.name,
            "short": a.short,
            "category": a.category,
            "baseScore": a.base_score,
            "enabled": True,
        }
        for a in _ALGOS
    ]


def run_all(ctx: dict, codes: list[str] | None = None) -> list[dict]:
    picked = codes or list(REGISTRY.keys())
    out = []
    for code in picked:
        algo = REGISTRY.get(code)
        if not algo:
            continue
        out.append(algo.run(ctx))
    return out
