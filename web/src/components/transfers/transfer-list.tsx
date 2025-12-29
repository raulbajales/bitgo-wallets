"use client";

import React, { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatCurrency, formatTimeAgo, truncateAddress } from "@/lib/utils";
import { type Transfer } from "./transfer-detail";

interface TransferListProps {
  transfers: Transfer[];
  loading?: boolean;
  onViewTransfer: (transfer: Transfer) => void;
  onRefresh: () => void;
}

export function TransferList({
  transfers,
  loading = false,
  onViewTransfer,
  onRefresh,
}: TransferListProps) {
  const [filter, setFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<"date" | "amount" | "status">("date");

  const getStatusVariant = (status: string) => {
    switch (status.toLowerCase()) {
      case "completed":
      case "confirmed":
        return "success";
      case "failed":
      case "rejected":
        return "destructive";
      case "pending_approval":
        return "warning";
      case "broadcast":
      case "submitted":
        return "default";
      default:
        return "outline";
    }
  };

  const filteredTransfers = transfers.filter((transfer) => {
    if (filter === "all") return true;
    return transfer.status === filter;
  });

  const sortedTransfers = [...filteredTransfers].sort((a, b) => {
    switch (sortBy) {
      case "date":
        return (
          new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
        );
      case "amount":
        return parseFloat(b.amountString) - parseFloat(a.amountString);
      case "status":
        return a.status.localeCompare(b.status);
      default:
        return 0;
    }
  });

  const getStatusCounts = () => {
    const counts = transfers.reduce((acc, transfer) => {
      acc[transfer.status] = (acc[transfer.status] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);
    return counts;
  };

  const statusCounts = getStatusCounts();

  if (loading && transfers.length === 0) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Loading transfers...</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header and Controls */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Transfers</h2>
          <p className="text-gray-600">
            Manage and track your transfer requests
          </p>
        </div>
        <Button onClick={onRefresh} variant="outline">
          {loading ? "Refreshing..." : "Refresh"}
        </Button>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="text-2xl font-bold">{transfers.length}</div>
            <div className="text-sm text-gray-500">Total Transfers</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-2xl font-bold text-yellow-600">
              {statusCounts.pending_approval || 0}
            </div>
            <div className="text-sm text-gray-500">Pending Approval</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-2xl font-bold text-blue-600">
              {statusCounts.broadcast || 0}
            </div>
            <div className="text-sm text-gray-500">In Progress</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="text-2xl font-bold text-green-600">
              {statusCounts.completed || 0}
            </div>
            <div className="text-sm text-gray-500">Completed</div>
          </CardContent>
        </Card>
      </div>

      {/* Filters and Sort */}
      <div className="flex flex-wrap gap-4 items-center bg-white p-4 rounded-lg border">
        <div className="flex items-center space-x-2">
          <span className="text-sm font-medium text-gray-700">Filter:</span>
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="text-sm border border-gray-300 rounded px-3 py-1 text-gray-900"
          >
            <option value="all">All Status</option>
            <option value="draft">Draft</option>
            <option value="pending_approval">Pending Approval</option>
            <option value="approved">Approved</option>
            <option value="broadcast">In Progress</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </div>

        <div className="flex items-center space-x-2">
          <span className="text-sm font-medium text-gray-700">Sort by:</span>
          <select
            value={sortBy}
            onChange={(e) =>
              setSortBy(e.target.value as "date" | "amount" | "status")
            }
            className="text-sm border border-gray-300 rounded px-3 py-1 text-gray-900"
          >
            <option value="date">Date</option>
            <option value="amount">Amount</option>
            <option value="status">Status</option>
          </select>
        </div>

        <div className="ml-auto text-sm text-gray-500">
          Showing {sortedTransfers.length} of {transfers.length} transfers
        </div>
      </div>

      {/* Transfer List */}
      <div className="space-y-4">
        {sortedTransfers.map((transfer) => (
          <Card key={transfer.id} className="hover:shadow-md transition-shadow">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center space-x-4">
                    {/* Amount and Coin */}
                    <div className="flex-shrink-0">
                      <div className="text-lg font-bold text-green-600">
                        {formatCurrency(transfer.amountString, transfer.coin)}
                      </div>
                      <div className="text-sm text-gray-500">
                        {transfer.coin.toUpperCase()}
                      </div>
                    </div>

                    {/* Transfer Details */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-2 mb-1">
                        <h3 className="font-medium text-gray-900 truncate">
                          To: {truncateAddress(transfer.recipientAddress)}
                        </h3>
                        <Badge variant={getStatusVariant(transfer.status)}>
                          {transfer.status.replace("_", " ").toUpperCase()}
                        </Badge>
                      </div>
                      <div className="text-sm text-gray-600 space-y-1">
                        <div>From: {transfer.walletLabel}</div>
                        <div className="flex items-center space-x-4">
                          <span>
                            Created:{" "}
                            {formatTimeAgo(new Date(transfer.createdAt))}
                          </span>
                          {transfer.memo && (
                            <span className="text-gray-400">
                              • {transfer.memo}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Approval Status */}
                    {transfer.requiredApprovals > 0 && (
                      <div className="text-center">
                        <div className="text-sm text-gray-500">Approvals</div>
                        <div className="font-medium">
                          {transfer.receivedApprovals}/
                          {transfer.requiredApprovals}
                        </div>
                        {transfer.status === "pending_approval" && (
                          <Badge variant="warning" className="mt-1 text-xs">
                            Needs approval
                          </Badge>
                        )}
                      </div>
                    )}

                    {/* Wallet Type */}
                    <div className="text-center">
                      <div className="text-sm text-gray-500">Type</div>
                      <Badge
                        variant={
                          transfer.transferType === "cold"
                            ? "secondary"
                            : "default"
                        }
                      >
                        {transfer.transferType === "custodial"
                          ? "Warm"
                          : transfer.transferType}
                      </Badge>
                    </div>

                    {/* Actions */}
                    <div className="flex-shrink-0">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onViewTransfer(transfer)}
                      >
                        View Details
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}

        {/* Empty State */}
        {sortedTransfers.length === 0 && !loading && (
          <Card>
            <CardContent className="text-center py-12">
              <div className="text-gray-400 mb-4">
                <svg
                  className="mx-auto h-16 w-16"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1}
                    d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-medium text-gray-900 mb-2">
                {filter === "all"
                  ? "No transfers found"
                  : `No ${filter} transfers`}
              </h3>
              <p className="text-gray-500 mb-6">
                {filter === "all"
                  ? "Create your first transfer to get started."
                  : "Try adjusting your filter to see more transfers."}
              </p>
              {filter !== "all" && (
                <Button variant="outline" onClick={() => setFilter("all")}>
                  Show All Transfers
                </Button>
              )}
            </CardContent>
          </Card>
        )}
      </div>

      {/* Load More (for pagination in real implementation) */}
      {sortedTransfers.length > 0 && (
        <div className="text-center">
          <Button variant="outline" disabled>
            Load More Transfers
          </Button>
        </div>
      )}
    </div>
  );
}
