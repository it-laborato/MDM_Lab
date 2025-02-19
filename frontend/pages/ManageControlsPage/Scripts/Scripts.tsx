import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import PATHS from "router/paths";
import scriptAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";
import { IScript } from "interfaces/script";

import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import InfoBanner from "components/InfoBanner";
import ScriptListHeading from "./components/ScriptListHeading";
import ScriptListItem from "./components/ScriptListItem";
import ScriptListPagination from "./components/ScriptListPagination";
import DeleteScriptModal from "./components/DeleteScriptModal";
import ScriptDetailsModal from "./components/ScriptDetailsModal";
import UploadList from "../components/UploadList";
import ScriptUploader from "./components/ScriptUploader";

const baseClass = "scripts";

const SCRIPTS_PER_PAGE = 10;

interface IScriptsProps {
  router: InjectedRouter; // v3
  teamIdForApi: number;
  currentPage: number;
}

const Scripts = ({ router, currentPage, teamIdForApi }: IScriptsProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  const [goBackToScriptDetails, setGoBackToScriptDetails] = useState(false); // Used for onCancel in delete modal

  const selectedScript = useRef<IScript | null>(null);

  const {
    data: { scripts, meta } = {},
    isLoading,
    isError,
    refetch: refetchScripts,
  } = useQuery<
    IScriptsResponse,
    AxiosError,
    IScriptsResponse,
    IListScriptsQueryKey[]
  >(
    [
      {
        scope: "scripts",
        team_id: teamIdForApi,
        page: currentPage,
        per_page: SCRIPTS_PER_PAGE,
      },
    ],
    ({ queryKey: [{ team_id, page, per_page }] }) =>
      scriptAPI.getScripts({ team_id, page, per_page }),
    {
      retry: false,
      refetchOnWindowFocus: false,
      staleTime: 3000,
    }
  );

  // pagination controls
  const path = PATHS.CONTROLS_SCRIPTS;
  const queryString = isPremiumTier ? `?team_id=${teamIdForApi}&` : "?";
  const onPrevPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage - 1}`));
  }, [router, path, currentPage, queryString]);
  const onNextPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage + 1}`));
  }, [router, path, currentPage, queryString]);

  const { config } = useContext(AppContext);
  if (!config) return null;

  const onClickScript = (script: IScript) => {
    selectedScript.current = script;
    setShowScriptDetailsModal(true);
  };

  const onCancelScriptDetails = () => {
    selectedScript.current = null;
    setShowScriptDetailsModal(false);
    setGoBackToScriptDetails(false);
  };

  const onClickDelete = (script: IScript) => {
    selectedScript.current = script;
    setShowDeleteScriptModal(true);
  };

  const onCancelDelete = () => {
    setShowDeleteScriptModal(false);

    if (goBackToScriptDetails) {
      setShowScriptDetailsModal(true);
    } else {
      selectedScript.current = null;
    }
  };

  const onDeleteScript = () => {
    selectedScript.current = null;
    setShowDeleteScriptModal(false);
    refetchScripts();
  };

  const onUploadScript = () => {
    refetchScripts();
  };

  const renderScriptsList = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (currentPage === 0 && !scripts?.length) {
      return null;
    }

    return (
      <>
        <UploadList
          keyAttribute="id"
          listItems={scripts || []}
          HeadingComponent={ScriptListHeading}
          ListItemComponent={({ listItem }) => (
            <ScriptListItem
              script={listItem}
              onDelete={onClickDelete}
              onClickScript={onClickScript}
            />
          )}
        />
        <ScriptListPagination
          meta={meta}
          isLoading={isLoading}
          onPrevPage={onPrevPage}
          onNextPage={onNextPage}
        />
      </>
    );
  };

  const renderScriptsDisabledBanner = () => (
    <InfoBanner color="yellow">
      <div>
        <b>Running scripts is disabled in organization settings.</b> You can
        still manage your library Windows scripts below.
      </div>
    </InfoBanner>
  );

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Upload scripts to remediate issues on Windows nodes.
        You can run scripts on individual nodes.{" "}
      </p>

      {config.server_settings.scripts_disabled && renderScriptsDisabledBanner()}
      {renderScriptsList()}
      <ScriptUploader currentTeamId={teamIdForApi} onUpload={onUploadScript} />
      {showDeleteScriptModal && selectedScript.current && (
        <DeleteScriptModal
          scriptName={selectedScript.current?.name}
          scriptId={selectedScript.current?.id}
          onCancel={onCancelDelete}
          onDone={onDeleteScript}
        />
      )}
      {showScriptDetailsModal && selectedScript.current && (
        <ScriptDetailsModal
          selectedScriptDetails={{
            script_id: selectedScript.current?.id,
            name: selectedScript.current?.name,
          }}
          onCancel={onCancelScriptDetails}
          onDelete={() => {
            setShowScriptDetailsModal(false);
            setShowDeleteScriptModal(true);
            setGoBackToScriptDetails(true);
          }}
          runScriptHelpText
        />
      )}
    </div>
  );
};

export default Scripts;
