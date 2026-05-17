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