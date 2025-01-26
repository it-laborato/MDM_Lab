import React, { useEffect, useState } from "react";
import { useQuery } from "react-query";

import { IApiError } from "interfaces/errors";
import { INode } from "interfaces/node";
import { INodeScript } from "interfaces/script";
import { IUser } from "interfaces/user";

import scriptsAPI, {
  INodeScriptsQueryKey,
  INodeScriptsResponse,
} from "services/entities/scripts";

import ScriptDetailsModal from "pages/ManageControlsPage/Scripts/components/ScriptDetailsModal";
import DeleteScriptModal from "pages/ManageControlsPage/Scripts/components/DeleteScriptModal";
import RunScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/RunScriptDetailsModal";
import RunScriptModal from "../RunScriptModal";

interface IScriptsProps {
  currentUser: IUser | null;
  node: INode;
  onCloseScriptModalGroup: () => void;
}

type ScriptGroupModals =
  | "run-script"
  | "view-script"
  | "run-script-details"
  | "delete-script"
  | null;

const ScriptModalGroup = ({
  currentUser,
  node,
  onCloseScriptModalGroup,
}: IScriptsProps) => {
  const [previousModal, setPreviousModal] = useState<ScriptGroupModals>(null);
  const [currentModal, setCurrentModal] = useState<ScriptGroupModals>(
    "run-script"
  );
  const [runScriptTablePage, setRunScriptTablePage] = useState(0);
  const [selectedScriptId, setSelectedScriptId] = useState<number | undefined>(
    undefined
  );
  const [selectedExecutionId, setSelectedExecutionId] = useState<
    string | undefined
  >(undefined);
  const [selectedScriptDetails, setSelectedScriptDetails] = useState<
    INodeScript | undefined
  >(undefined);
  // This sets a loading state of the Run script modal during a run request
  const [runScriptRequested, setRunScriptRequested] = useState(false);

  // Almost everything from this is needed on RunScript.tsx modal
  // except refetch is used multiple places
  const {
    data: runScriptTableResponse,
    isError,
    isLoading,
    isFetching,
    refetch: refetchNodeScripts,
  } = useQuery<
    INodeScriptsResponse,
    IApiError,
    INodeScriptsResponse,
    INodeScriptsQueryKey[]
  >(
    [
      {
        scope: "node_scripts",
        node_id: node.id,
        page: runScriptTablePage,
        per_page: 10,
      },
    ],
    ({ queryKey }) => scriptsAPI.getNodeScripts(queryKey[0]),
    {
      refetchOnWindowFocus: false,
      retry: false,
      staleTime: 3000,
      onSuccess: () => {
        setRunScriptRequested(false);
      },
    }
  );

  // Note: Script metadata and script content require two separate API calls
  // Source: https://mdmlabdm.com/docs/rest-api/rest-api#example-get-script
  // So to get script name, we pass it into this modal instead of another API call
  // If in future iterations we want more script metadata, call scriptAPI.getScript()
  // and consider refactoring .getScript to return script content as well
  const {
    data: selectedScriptContent,
    error: isSelectedScriptContentError,
    isLoading: isLoadingSelectedScriptContent,
  } = useQuery<string, Error>(
    ["scriptContent", selectedScriptId],
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    () => scriptsAPI.downloadScript(selectedScriptId!),
    {
      refetchOnWindowFocus: false,
      enabled: !!selectedScriptId,
    }
  );

  // Anytime a script runs, return back to Run script modal
  useEffect(() => {
    if (runScriptRequested) {
      setCurrentModal("run-script");
    }
  }, [runScriptRequested]);

  return (
    <>
      <RunScriptModal
        currentUser={currentUser}
        node={node}
        onClose={onCloseScriptModalGroup}
        onClickViewScript={(scriptId: number, scriptDetails: INodeScript) => {
          setPreviousModal(currentModal);
          setCurrentModal("view-script");
          setSelectedScriptId(scriptId);
          setSelectedScriptDetails(scriptDetails);
        }}
        onClickRunDetails={(scriptExecutionId: string) => {
          setPreviousModal(currentModal);
          setCurrentModal("run-script-details");
          scriptExecutionId && setSelectedExecutionId(scriptExecutionId);
        }}
        runScriptRequested={runScriptRequested}
        refetchNodeScripts={refetchNodeScripts}
        page={runScriptTablePage}
        setPage={setRunScriptTablePage}
        nodeScriptResponse={runScriptTableResponse}
        isFetching={isFetching}
        isLoading={isLoading}
        isError={isError}
        setRunScriptRequested={setRunScriptRequested}
        isHidden={currentModal !== "run-script"}
      />

      <ScriptDetailsModal
        nodeId={node.id}
        nodeTeamId={node.team_id}
        selectedScriptDetails={selectedScriptDetails}
        selectedScriptContent={selectedScriptContent}
        onCancel={() => {
          setCurrentModal(previousModal);
          setPreviousModal(null);
        }}
        onDelete={() => {
          setPreviousModal(currentModal);
          setCurrentModal("delete-script");
        }}
        onClickRunDetails={(scriptExecutionId: string) => {
          setPreviousModal(currentModal);
          setCurrentModal("run-script-details");
          scriptExecutionId && setSelectedExecutionId(scriptExecutionId);
        }}
        setRunScriptRequested={setRunScriptRequested}
        refetchNodeScripts={refetchNodeScripts}
        isLoadingScriptContent={isLoadingSelectedScriptContent}
        isScriptContentError={isSelectedScriptContentError}
        isHidden={currentModal !== "view-script"}
        showNodeScriptActions
      />
      <DeleteScriptModal
        scriptId={selectedScriptDetails?.script_id || 1}
        scriptName={selectedScriptDetails?.name || ""}
        onCancel={() => {
          setCurrentModal(previousModal);
          setPreviousModal("run-script");
        }}
        onDone={() => {
          // The delete API call is handled in DeleteScriptModal
          setCurrentModal(previousModal);
          setPreviousModal("run-script");
          refetchNodeScripts();
          setSelectedScriptDetails(undefined);
        }}
        isHidden={currentModal !== "delete-script"}
      />
      <RunScriptDetailsModal
        scriptExecutionId={selectedExecutionId || ""}
        onCancel={() => {
          if (previousModal === "view-script") {
            setCurrentModal(previousModal);
            setPreviousModal("run-script");
          } else if (previousModal === "run-script") {
            setCurrentModal(previousModal);
            setPreviousModal(null);
          }
        }}
        isHidden={currentModal !== "run-script-details"}
      />
    </>
  );
};

export default ScriptModalGroup;
