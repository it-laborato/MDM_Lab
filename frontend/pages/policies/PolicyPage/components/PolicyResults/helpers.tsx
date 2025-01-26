import { IPolicyNodeResponse } from "interfaces/node";

export const getYesNoCounts = (nodeResponses: IPolicyNodeResponse[]) => {
  const yesNoCounts = nodeResponses.reduce(
    (acc, nodeResponse) => {
      if (nodeResponse.query_results?.length) {
        acc.yes += 1;
      } else {
        acc.no += 1;
      }
      return acc;
    },
    { yes: 0, no: 0 }
  );

  return yesNoCounts;
};

export default { getYesNoCounts };
