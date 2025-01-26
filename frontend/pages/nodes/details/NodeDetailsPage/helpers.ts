import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "refetch error.";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown, nodeName: string) => {
  let errorMessage = getErrorReason(e, {
    reasonIncludes: "Node does not have MDM turned on",
  });

  if (!errorMessage) {
    errorMessage = DEFAULT_ERROR_MESSAGE;
  }

  return `Node "${nodeName}" ${errorMessage}`;
};
