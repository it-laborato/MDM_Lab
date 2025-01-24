import Button from "components/buttons/Button";
import React from "react";
import { IScriptsResponse } from "services/entities/scripts";

// @ts-ignore
import MdmlabIcon from "components/icons/MdmlabIcon";

const baseClass = "script-list-pagination";

interface IScriptsListPaginationProps {
  meta: IScriptsResponse["meta"] | undefined;
  isLoading: boolean;
  onPrevPage: () => void;
  onNextPage: () => void;
}

const ScriptsListPagination = ({
  meta,
  isLoading,
  onPrevPage,
  onNextPage,
}: IScriptsListPaginationProps) => {
  return (
    <div className={baseClass}>
      <Button
        disabled={isLoading || !meta?.has_previous_results}
        onClick={onPrevPage}
        variant="unstyled"
        className={`${baseClass}__button`}
      >
        <>
          <MdmlabIcon name="chevronleft" /> Previous
        </>
      </Button>
      <Button
        disabled={isLoading || !meta?.has_next_results}
        onClick={onNextPage}
        variant="unstyled"
        className={`${baseClass}__button`}
      >
        <>
          Next <MdmlabIcon name="chevronright" />
        </>
      </Button>
    </div>
  );
};

export default ScriptsListPagination;
