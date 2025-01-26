const DEFAULT_ERROR_MESSAGE = "refetch error.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown, nodeName: string) => {
  return `Node "${nodeName}" ${DEFAULT_ERROR_MESSAGE}`;
};
