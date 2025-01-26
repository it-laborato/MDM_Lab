import React, { useRef, useEffect, useState } from "react";
import { Row } from "react-table";
import { isEmpty, pullAllBy } from "lodash";

import { INode } from "interfaces/node";
import { HOSTS_SEARCH_BOX_PLACEHOLDER } from "utilities/constants";

import DataError from "components/DataError";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon/InputFieldWithIcon";
import TableContainer from "components/TableContainer";
import { ITargestInputNodeTableConfig } from "./TargetsInputNodesTableConfig";

interface ITargetsInputProps {
  tabIndex?: number;
  searchText: string;
  searchResults: INode[];
  isTargetsLoading: boolean;
  hasFetchError: boolean;
  targetedNodes: INode[];
  searchResultsTableConfig: ITargestInputNodeTableConfig[];
  selectedNodesTableConifg: ITargestInputNodeTableConfig[];
  /** disabled pagination for the results table. The pagination is currently
   * client side pagination. Defaults to `false` */
  disablePagination?: boolean;
  label?: string;
  placeholder?: string;
  autofocus?: boolean;
  setSearchText: (value: string) => void;
  handleRowSelect: (value: Row<INode>) => void;
}

const baseClass = "targets-input";

const DEFAULT_LABEL = "Target specific nodes";

const TargetsInput = ({
  tabIndex,
  searchText,
  searchResults,
  isTargetsLoading,
  hasFetchError,
  targetedNodes,
  searchResultsTableConfig,
  selectedNodesTableConifg,
  disablePagination = false,
  label = DEFAULT_LABEL,
  placeholder = HOSTS_SEARCH_BOX_PLACEHOLDER,
  autofocus = false,
  handleRowSelect,
  setSearchText,
}: ITargetsInputProps): JSX.Element => {
  const dropdownRef = useRef<HTMLDivElement | null>(null);
  const dropdownNodes =
    searchResults && pullAllBy(searchResults, targetedNodes, "display_name");

  const [isActiveSearch, setIsActiveSearch] = useState(false);

  const isSearchError = !isEmpty(searchText) && hasFetchError;

  // Closes target search results when clicking outside of results
  // But not during API loading state as it will reopen on API return
  useEffect(() => {
    if (!isTargetsLoading) {
      const handleClickOutside = (event: MouseEvent) => {
        if (
          dropdownRef.current &&
          !dropdownRef.current.contains(event.target as Node)
        ) {
          setIsActiveSearch(false);
        }
      };

      document.addEventListener("mousedown", handleClickOutside);
      return () => {
        document.removeEventListener("mousedown", handleClickOutside);
      };
    }
  }, [isTargetsLoading]);

  useEffect(() => {
    setIsActiveSearch(
      !isEmpty(searchText) && (!hasFetchError || isTargetsLoading)
    );
  }, [searchText, hasFetchError, isTargetsLoading]);
  return (
    <div>
      <div className={baseClass}>
        <InputFieldWithIcon
          autofocus={autofocus}
          type="search"
          iconSvg="search"
          value={searchText}
          iconPosition="start"
          label={label}
          placeholder={placeholder}
          onChange={setSearchText}
        />
        {isActiveSearch && (
          <div
            className={`${baseClass}__nodes-search-dropdown`}
            ref={dropdownRef}
          >
            <TableContainer<Row<INode>>
              columnConfigs={searchResultsTableConfig}
              data={dropdownNodes}
              isLoading={isTargetsLoading}
              emptyComponent={() => (
                <div className="empty-search">
                  <div className="empty-search__inner">
                    <h4>No matching nodes.</h4>
                    <p>
                      Expecting to see nodes? Try again in a few seconds as the
                      system catches up.
                    </p>
                  </div>
                </div>
              )}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              disableCount
              disablePagination
              disableMultiRowSelect
              onClickRow={handleRowSelect}
              keyboardSelectableRows
            />
          </div>
        )}
        {isSearchError && (
          <div className={`${baseClass}__nodes-search-dropdown`}>
            <DataError />
          </div>
        )}
        <div className={`${baseClass}__nodes-selected-table`}>
          <TableContainer
            columnConfigs={selectedNodesTableConifg}
            data={targetedNodes}
            isLoading={false}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            disablePagination={disablePagination}
            isClientSidePagination={!disablePagination}
            emptyComponent={() => <></>}
          />
        </div>
      </div>
    </div>
  );
};

export default TargetsInput;
