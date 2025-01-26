import React, { useEffect, useState } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { useDebouncedCallback } from "use-debounce";

import { INode } from "interfaces/node";
import targetsAPI, { ITargetsSearchResponse } from "services/entities/targets";

import TargetsInput from "components/LiveQuery/TargetsInput";

import LabelForm from "../LabelForm";
import { ILabelFormData } from "../LabelForm/LabelForm";
import { generateTableHeaders } from "./LabelNodeTargetTableConfig";

const baseClass = "ManualLabelForm";

export const LABEL_TARGET_HOSTS_INPUT_LABEL = "Select nodes";
const LABEL_TARGET_HOSTS_INPUT_PLACEHOLDER =
  "Search name, nodename, or serial number";
const DEBOUNCE_DELAY = 500;

export interface IManualLabelFormData {
  name: string;
  description: string;
  targetedNodes: INode[];
}

interface ITargetsQueryKey {
  scope: string;
  query?: string | null;
  excludedNodeIds?: number[];
}

interface IManualLabelFormProps {
  defaultName?: string;
  defaultDescription?: string;
  defaultTargetedNodes?: INode[];
  onSave: (formData: IManualLabelFormData) => void;
  onCancel: () => void;
}

const ManualLabelForm = ({
  defaultName = "",
  defaultDescription = "",
  defaultTargetedNodes = [],
  onSave,
  onCancel,
}: IManualLabelFormProps) => {
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState("");
  const [isDebouncing, setIsDebouncing] = useState(false);
  const [targetedNodes, setTargetedNodes] = useState<INode[]>(
    defaultTargetedNodes
  );

  const targetdNodesIds = targetedNodes.map((node) => node.id);

  const debounceSearch = useDebouncedCallback(
    (search: string) => {
      setDebouncedSearchQuery(search);
      setIsDebouncing(false);
    },
    DEBOUNCE_DELAY,
    { trailing: true }
  );

  // TODO: find a better way to debounce search requests
  useEffect(() => {
    setIsDebouncing(true);
    debounceSearch(searchQuery);
  }, [debounceSearch, searchQuery]);

  const {
    data: nodeTargets,
    isLoading: isLoadingSearchResults,
    isError: isErrorSearchResults,
  } = useQuery<ITargetsSearchResponse, Error, INode[], ITargetsQueryKey[]>(
    [
      {
        scope: "labels-targets-search",
        query: debouncedSearchQuery,
        excludedNodeIds: targetdNodesIds,
      },
    ],
    ({ queryKey }) => {
      const { query, excludedNodeIds } = queryKey[0];
      return targetsAPI.search({
        query: query ?? "",
        excluded_node_ids: excludedNodeIds ?? null,
      });
    },
    {
      select: (data) => data.nodes,
      enabled: searchQuery !== "",
    }
  );

  const onNodeSelect = (row: Row<INode>) => {
    setTargetedNodes((prevNodes) => prevNodes.concat(row.original));
    setSearchQuery("");
  };

  const onNodeRemove = (row: Row<INode>) => {
    setTargetedNodes((prevNodes) =>
      prevNodes.filter((h) => h.id !== row.original.id)
    );
  };

  const onSaveNewLabel = (
    labelFormData: ILabelFormData,
    labelFormDataValid: boolean
  ) => {
    if (labelFormDataValid) {
      // values from LabelForm component must be valid too
      onSave({ ...labelFormData, targetedNodes });
    }
  };

  const onChangeSearchQuery = (value: string) => {
    setSearchQuery(value);
  };

  const resultsTableConfig = generateTableHeaders();
  const selectedNodesTableConfig = generateTableHeaders(onNodeRemove);

  return (
    <div className={baseClass}>
      <LabelForm
        defaultName={defaultName}
        defaultDescription={defaultDescription}
        onCancel={onCancel}
        onSave={onSaveNewLabel}
        additionalFields={
          <TargetsInput
            label={LABEL_TARGET_HOSTS_INPUT_LABEL}
            placeholder={LABEL_TARGET_HOSTS_INPUT_PLACEHOLDER}
            searchText={searchQuery}
            searchResultsTableConfig={resultsTableConfig}
            selectedNodesTableConifg={selectedNodesTableConfig}
            isTargetsLoading={isLoadingSearchResults || isDebouncing}
            hasFetchError={isErrorSearchResults}
            searchResults={nodeTargets ?? []}
            targetedNodes={targetedNodes}
            setSearchText={onChangeSearchQuery}
            handleRowSelect={onNodeSelect}
          />
        }
      />
    </div>
  );
};

export default ManualLabelForm;
