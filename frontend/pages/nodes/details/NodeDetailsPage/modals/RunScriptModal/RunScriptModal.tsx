import React, { useCallback, useContext, useMemo } from "react";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";
import { INode } from "interfaces/node";
import { INodeScript } from "interfaces/script";
import { IUser } from "interfaces/user";

import scriptsAPI, { INodeScriptsResponse } from "services/entities/scripts";

import Button from "components/buttons/Button";
import DataError from "components/DataError/DataError";
import EmptyTable from "components/EmptyTable";
import Modal from "components/Modal";
import Spinner from "components/Spinner/Spinner";

import TableContainer, {
  ITableQueryData,
} from "components/TableContainer/TableContainer";

import { generateTableColumnConfigs } from "./ScriptsTableConfig";

const baseClass = "run-script-modal";

interface IRunScriptModalProps {
  currentUser: IUser | null;
  node: INode;
  onClose: () => void;
  runScriptRequested: boolean;
  refetchNodeScripts: () => void;
  page: number;
  setPage: React.Dispatch<React.SetStateAction<number>>;
  nodeScriptResponse?: INodeScriptsResponse;
  isFetching: boolean;
  isLoading: boolean;
  isError: boolean;
  setRunScriptRequested: React.Dispatch<React.SetStateAction<boolean>>;
  onClickViewScript: (scriptId: number, scriptDetails: INodeScript) => void;
  onClickRunDetails: (scriptExecutionId: string) => void;
  isHidden: boolean;
}

const EmptyComponent = () => <></>;

const RunScriptModal = ({
  currentUser,
  node,
  onClose,
  runScriptRequested,
  refetchNodeScripts,
  page,
  setPage,
  setRunScriptRequested,
  nodeScriptResponse,
  isFetching,
  isLoading,
  isError,
  onClickViewScript,
  onClickRunDetails,
  isHidden = false,
}: IRunScriptModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const onSelectAction = useCallback(
    async (action: string, script: INodeScript) => {
      switch (action) {
        case "showRunDetails": {
          script.last_execution?.execution_id &&
            onClickRunDetails(script.last_execution?.execution_id);
          break;
        }
        case "run": {
          try {
            setRunScriptRequested(true);
            await scriptsAPI.runScript({
              node_id: node.id,
              script_id: script.script_id,
            });
            renderFlash(
              "success",
              "Script is running or will run when the node comes online."
            );
            refetchNodeScripts();
          } catch (e) {
            renderFlash("error", getErrorReason(e));
            setRunScriptRequested(false);
          }
          break;
        }
        default: // do nothing
      }
    },
    [
      node.id,
      onClickRunDetails,
      refetchNodeScripts,
      renderFlash,
      setRunScriptRequested,
    ]
  );

  const onQueryChange = useCallback(({ pageIndex }: ITableQueryData) => {
    setPage(pageIndex);
  }, []);

  const scriptColumnConfigs = useMemo(
    () =>
      generateTableColumnConfigs(
        currentUser,
        node.team_id,
        !!config?.server_settings?.scripts_disabled,
        onClickViewScript,
        onSelectAction
      ),
    [currentUser, node.team_id, config, onClickViewScript, onSelectAction]
  );

  if (!config) return null;

  const tableData = nodeScriptResponse?.scripts;

  return (
    <Modal
      title="Run script"
      onExit={onClose}
      onEnter={onClose}
      className={`${baseClass}`}
      isLoading={runScriptRequested || isFetching || isLoading}
      isHidden={isHidden}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          {isLoading && <Spinner />}
          {!isLoading && isError && <DataError />}
          {!isLoading && !isError && (!tableData || tableData.length === 0) && (
            <EmptyTable
              header="No scripts available for this node"
              info="Expecting to see scripts? Close this modal and try again."
            />
          )}
          {!isLoading && !isError && tableData && tableData.length > 0 && (
            <TableContainer
              resultsTitle=""
              emptyComponent={EmptyComponent}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              columnConfigs={scriptColumnConfigs}
              data={tableData}
              isLoading={runScriptRequested || isFetching}
              onQueryChange={onQueryChange}
              disableNextPage={!nodeScriptResponse?.meta.has_next_results}
              defaultPageIndex={page}
              pageSize={10}
              disableCount
              disableTableHeader
            />
          )}
        </div>
        <div className={`modal-cta-wrap`}>
          <Button onClick={onClose} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default React.memo(RunScriptModal);
