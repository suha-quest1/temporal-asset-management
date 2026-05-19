export const startCapitalCall = async (payload: any) => {
  const response = await fetch('/api/capital-calls', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`Failed to start capital call: ${errorBody}`);
  }

  return response.json();
};

export const getCapitalCalls = async () => {
  const response = await fetch('/api/capital-calls');
  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`Failed to fetch capital calls: ${errorBody}`);
  }
  return response.json();
};

export const getRiskyLPs = async () => {
  const response = await fetch('/api/capital-calls/risky-lps');
  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`Failed to fetch risky LPs: ${errorBody}`);
  }
  return response.json();
};

export const postGPDecision = async (callId: string, payload: { lpId: string, action: string, gpName?: string }) => {
  const response = await fetch(`/api/capital-calls/${callId}/gp-decision`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`Failed to post GP decision: ${errorBody}`);
  }

  return response.json();
};

export const getCallLPs = async (callId: string) => {
  const response = await fetch(`/api/capital-calls/${callId}/lps`);
  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`Failed to fetch call LPs: ${errorBody}`);
  }
  return response.json();
};

export const postLPResponse = async (callId: string, payload: { lpId: string, amount: number }) => {
  const response = await fetch(`/api/capital-calls/${callId}/lp-response`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`Failed to post LP response: ${errorBody}`);
  }

  return response.json();
};