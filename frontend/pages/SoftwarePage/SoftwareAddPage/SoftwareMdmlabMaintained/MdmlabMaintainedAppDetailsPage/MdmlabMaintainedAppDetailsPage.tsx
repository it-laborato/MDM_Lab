import React, { useContext, useState } from "react";
import { Location } from "history";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { useErrorHandler } from "react-error-boundary";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import softwareAPI from "services/entities/software";
import teamPoliciesAPI from "services/entities/team_policies";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import { QueryContext } from "context/query";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { Platform, PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import { ILabelSummary } from "interfaces/label";
import useToggleSidePanel from "hooks/useToggleSidePanel";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Card from "components/Card";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import MdmlabAppDetailsForm from "./MdmlabAppDetailsForm";
import { IMdmlabMaintainedAppFormData } from "./MdmlabAppDetailsForm/MdmlabAppDetailsForm";
import AddMdmlabAppSoftwareModal from "./AddMdmlabAppSoftwareModal";

import {
  getErrorMessage,
  getMdmlabAppPolicyDescription,
  getMdmlabAppPolicyName,
  getMdmlabAppPolicyQuery,
} from "./helpers";

const baseClass = "mdmlab-maintained-app-details-page";

interface ISoftwareSummaryProps {
  name: string;
  platform: string;
  version: string;
}

const MdmlabAppSummary = ({
  name,
  platform,
  version,
}: ISoftwareSummaryProps) => {
  return (
    <Card
      className={`${baseClass}__mdmlab-app-summary`}
      borderRadiusSize="medium"
    >
      <SoftwareIcon name={name} size="medium" />
      <div className={`${baseClass}__mdmlab-app-summary--details`}>
        <div className={`${baseClass}__mdmlab-app-summary--title`}>{name}</div>
        <div className={`${baseClass}__mdmlab-app-summary--info`}>
          <div className={`${baseClass}__mdmlab-app-summary--details--platform`}>
            {PLATFORM_DISPLAY_NAMES[platform as Platform]}
          </div>
          &bull;
          <div className={`${baseClass}__mdmlab-app-summary--details--version`}>
            {version}
          </div>
        </div>
      </div>
    </Card>
  );
};

export interface IMdmlabMaintainedAppDetailsQueryParams {
  team_id?: string;
}

interface IMdmlabMaintainedAppDetailsRouteParams {
  id: string;
}

interface IMdmlabMaintainedAppDetailsPageProps {
  location: Location<IMdmlabMaintainedAppDetailsQueryParams>;
  router: InjectedRouter;
  routeParams: IMdmlabMaintainedAppDetailsRouteParams;
}

/** This type includes the editable form data as well as the mdmlab maintained
 * app id */
export type IAddMdmlabMaintainedData = IMdmlabMaintainedAppFormData & {
  appId: number;
};

const MdmlabMaintainedAppDetailsPage = ({
  location,
  router,
  routeParams,
}: IMdmlabMaintainedAppDetailsPageProps) => {
  const teamId = location.query.team_id;
  const appId = parseInt(routeParams.id, 10);
  if (isNaN(appId)) {
    router.push(PATHS.SOFTWARE_ADD_MDMLAB_MAINTAINED);
  }

  const { renderFlash } = useContext(NotificationContext);
  const handlePageError = useErrorHandler();
  const { isPremiumTier } = useContext(AppContext);
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(false);
  const [
    showAddMdmlabAppSoftwareModal,
    setShowAddMdmlabAppSoftwareModal,
  ] = useState(false);

  const {
    data: mdmlabApp,
    isLoading: isLoadingMdmlabApp,
    isError: isErrorMdmlabApp,
  } = useQuery(
    ["mdmlab-maintained-app", appId],
    () => softwareAPI.getMdmlabMaintainedApp(appId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
      retry: false,
      select: (res) => res.mdmlab_maintained_app,
      onError: (error) => handlePageError(error),
    }
  );

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary().then((res) => getCustomLabels(res.labels)),

    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
      staleTime: 10000,
    }
  );

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const backToAddSoftwareUrl = `${
    PATHS.SOFTWARE_ADD_MDMLAB_MAINTAINED
  }?${buildQueryStringFromParams({ team_id: teamId })}`;

  const onCancel = () => {
    router.push(backToAddSoftwareUrl);
  };

  const onSubmit = async (formData: IMdmlabMaintainedAppFormData) => {
    // this should not happen but we need to handle the type correctly
    if (!teamId) return;

    setShowAddMdmlabAppSoftwareModal(true);

    const { installType } = formData;
    let titleId: number | undefined;
    try {
      const res = await softwareAPI.addMdmlabMaintainedApp(
        parseInt(teamId, 10),
        {
          ...formData,
          appId,
        }
      );
      titleId = res.software_title_id;

      // for manual install we redirect only on a successful software add.
      if (installType === "manual") {
        router.push(
          `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
            team_id: teamId,
            available_for_install: true,
          })}`
        );
        renderFlash(
          "success",
          <>
            <b>{mdmlabApp?.name}</b> successfully added.
          </>
        );
      }
    } catch (error) {
      // quick exit if there was an error adding the software. Skip the policy
      // creation.
      renderFlash("error", getErrorMessage(error));
      setShowAddMdmlabAppSoftwareModal(false);
      return;
    }

    // If the install type is automatic we now need to create the new policy.
    if (installType === "automatic" && mdmlabApp) {
      try {
        await teamPoliciesAPI.create({
          name: getMdmlabAppPolicyName(mdmlabApp.name),
          description: getMdmlabAppPolicyDescription(mdmlabApp.name),
          query: getMdmlabAppPolicyQuery(mdmlabApp.name),
          team_id: parseInt(teamId, 10),
          software_title_id: titleId,
          platform: "darwin",
        });

        renderFlash(
          "success",
          <>
            <b>{mdmlabApp?.name}</b> successfully added.
          </>,
          { persistOnPageChange: true }
        );
      } catch (e) {
        renderFlash(
          "error",
          "Couldn't add automatic install policy. Software is successfully added. To retry, delete software and add it again.",
          { persistOnPageChange: true }
        );
      }

      // for automatic install we redirect on both a successful and error policy
      // add because the software was already successfuly added.
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
          team_id: teamId,
          available_for_install: true,
        })}`
      );
    }

    setShowAddMdmlabAppSoftwareModal(false);
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (isLoadingMdmlabApp || isLoadingLabels) {
      return <Spinner />;
    }

    if (isErrorMdmlabApp || isErrorLabels) {
      return <DataError className={`${baseClass}__data-error`} />;
    }

    if (mdmlabApp) {
      return (
        <>
          <BackLink
            text="Back to add software"
            path={backToAddSoftwareUrl}
            className={`${baseClass}__back-to-add-software`}
          />
          <h1>{mdmlabApp.name}</h1>
          <div className={`${baseClass}__page-content`}>
            <MdmlabAppSummary
              name={mdmlabApp.name}
              platform={mdmlabApp.platform}
              version={mdmlabApp.version}
            />
            <MdmlabAppDetailsForm
              labels={labels || []}
              name={mdmlabApp.name}
              showSchemaButton={!isSidePanelOpen}
              defaultInstallScript={mdmlabApp.install_script}
              defaultPostInstallScript={mdmlabApp.post_install_script}
              defaultUninstallScript={mdmlabApp.uninstall_script}
              onClickShowSchema={() => setSidePanelOpen(true)}
              onCancel={onCancel}
              onSubmit={onSubmit}
            />
          </div>
        </>
      );
    }

    return null;
  };

  return (
    <>
      <MainContent className={baseClass}>
        <>{renderContent()}</>
      </MainContent>
      {isPremiumTier && mdmlabApp && isSidePanelOpen && (
        <SidePanelContent className={`${baseClass}__side-panel`}>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={() => setSidePanelOpen(false)}
          />
        </SidePanelContent>
      )}
      {showAddMdmlabAppSoftwareModal && <AddMdmlabAppSoftwareModal />}
    </>
  );
};

export default MdmlabMaintainedAppDetailsPage;
