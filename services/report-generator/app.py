"""Report Generator — produces JSON (and optionally PDF) liquidity reports.

Receives the full CapitalCallResult and writes a report to /reports/.
"""

import json
import os
from datetime import datetime, timezone
from flask import Flask, request, jsonify

app = Flask(__name__)

REPORT_DIR = os.environ.get("REPORT_DIR", "/reports")


@app.route("/generate", methods=["POST"])
def generate():
    data = request.get_json(force=True)
    call_id = data.get("callId", "unknown")
    fund_id = data.get("fundId", "unknown")
    call_result = data.get("callResult", {})

    report = {
        "reportType": "LiquidityReport",
        "generatedAt": datetime.now(timezone.utc).isoformat(),
        "callId": call_id,
        "fundId": fund_id,
        "summary": {
            "targetAmountUSD": call_result.get("targetAmountUSD", 0),
            "totalCommitted": call_result.get("totalCommitted", 0),
            "gapUSD": call_result.get("gapUSD", 0),
            "gapPercent": call_result.get("gapPercent", 0),
            "bridgeTriggered": call_result.get("bridgeTriggered", False),
        },
        "lpDetails": call_result.get("lpResponses", []),
        "portfolioRisk": call_result.get("portfolioRisk", {}),
        "bridgeResult": call_result.get("bridgeResult"),
    }

    os.makedirs(REPORT_DIR, exist_ok=True)

    # Write JSON report
    json_path = os.path.join(REPORT_DIR, f"{call_id}.json")
    with open(json_path, "w") as f:
        json.dump(report, f, indent=2)

    app.logger.info("Report generated: %s", json_path)

    return jsonify({
        "reportPath": f"/reports/{call_id}.json"
    })


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"service": "report-generator", "status": "ok"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5002)
