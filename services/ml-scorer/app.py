"""ML Scorer — predicts LP default risk using synthetic scoring logic.

In production this would load a trained scikit-learn model.  For the demo
it computes a deterministic score from LP attributes so the outcome is
reproducible.
"""

import hashlib
import math
from flask import Flask, request, jsonify

app = Flask(__name__)


def _synthetic_score(lp_id: str, status: str, amount: float) -> float:
    """Return a deterministic risk score between 0.0 and 1.0.

    - LPs whose ID ends with "08" are flagged as high-risk (> 0.7)
      to exercise the GP-escalation path in the demo.
    - Defaulted LPs get a high base score.
    - Everyone else gets a hash-derived score in [0.05, 0.55].
    """
    if lp_id.endswith("08"):
        return 0.82

    if status == "defaulted":
        return 0.75 + (hash(lp_id) % 20) / 100  # 0.75 – 0.94

    # Deterministic but varied score for normal LPs
    digest = hashlib.sha256(lp_id.encode()).hexdigest()
    raw = int(digest[:8], 16) / 0xFFFFFFFF  # 0.0 – 1.0
    return round(0.05 + raw * 0.50, 4)       # 0.05 – 0.55


@app.route("/score", methods=["POST"])
def score():
    data = request.get_json(force=True)
    lp_responses = data.get("lpResponses", [])

    scores = []
    for lp in lp_responses:
        lp_id = lp.get("lpId", "")
        status = lp.get("status", "")
        amount = lp.get("amountUSD", 0)
        risk = _synthetic_score(lp_id, status, amount)
        scores.append({"lpId": lp_id, "riskScore": risk})
        app.logger.info("Scored %s → %.4f (status=%s)", lp_id, risk, status)

    return jsonify(scores)


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"service": "ml-scorer", "status": "ok"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5001)
