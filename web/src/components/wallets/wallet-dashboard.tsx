"use client";

import React, { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { WalletCard, type Wallet } from "@/components/wallets/wallet-card";

export function WalletDashboard() {
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [syncing, setSyncing] = useState<string | null>(null);

  // Mock data for development - replace with actual API calls
  useEffect(() => {
    // Simulate API call
    const mockWallets: Wallet[] = [
      {
        id: "1",
        bitgoWalletId: "64a5b2c8e9f1a2b3c4d5e6f7",
        label: "Main Bitcoin Wallet",
        coin: "btc",
        walletType: "custodial",
        balanceString: "1.25430000",
        confirmedBalanceString: "1.25430000",
        spendableBalanceString: "1.20000000",
        isActive: true,
        frozen: false,
        tags: ["production", "main"],
        createdAt: "2024-01-15T10:00:00Z",
        updatedAt: "2024-01-20T14:30:00Z",
      },
      {
        id: "2",
        bitgoWalletId: "74b6c3d9faeafcd4e5f6g8h9",
        label: "Ethereum Treasury",
        coin: "eth",
        walletType: "cold",
        balanceString: "45.75000000",
        confirmedBalanceString: "45.75000000",
        spendableBalanceString: "40.00000000",
        isActive: true,
        frozen: false,
        tags: ["treasury", "cold-storage"],
        createdAt: "2024-01-10T08:00:00Z",
        updatedAt: "2024-01-19T16:45:00Z",
      },
      {
        id: "3",
        bitgoWalletId: "84c7d4eafbfbade5f6g7h9i0",
        label: "USDC Operations",
        coin: "usdc",
        walletType: "hot",
        balanceString: "50000.000000",
        confirmedBalanceString: "50000.000000",
        spendableBalanceString: "48500.000000",
        isActive: true,
        frozen: false,
        tags: ["operations", "stablecoin"],
        createdAt: "2024-01-12T12:00:00Z",
        updatedAt: "2024-01-21T09:15:00Z",
      },
    ];

    setTimeout(() => {
      setWallets(mockWallets);
      setLoading(false);
    }, 1000);
  }, []);

  const handleDiscoverWallets = async () => {
    try {
      setLoading(true);
      // TODO: Implement actual API call
      // const response = await fetch('/api/v1/wallets/discover')
      // const data = await response.json()

      // Mock discovery
      setTimeout(() => {
        setLoading(false);
        // Could add newly discovered wallets to the list
      }, 2000);
    } catch (err) {
      setError("Failed to discover wallets");
      setLoading(false);
    }
  };

  const handleViewDetails = (wallet: Wallet) => {
    // TODO: Navigate to wallet detail page
    console.log("View details for wallet:", wallet.id);
  };

  const handleCreateTransfer = (wallet: Wallet) => {
    // TODO: Navigate to transfer creation form
    console.log("Create transfer for wallet:", wallet.id);
  };

  const handleSyncBalance = async (wallet: Wallet) => {
    try {
      setSyncing(wallet.id);
      // TODO: Implement actual API call
      // await fetch(`/api/v1/wallets/${wallet.id}/sync-balance`, { method: 'POST' })

      // Mock sync
      setTimeout(() => {
        setSyncing(null);
        // Update wallet balances
        setWallets((prev) =>
          prev.map((w) =>
            w.id === wallet.id
              ? { ...w, updatedAt: new Date().toISOString() }
              : w
          )
        );
      }, 1500);
    } catch (err) {
      setError("Failed to sync wallet balance");
      setSyncing(null);
    }
  };

  const getTotalBalance = () => {
    return wallets.reduce((total, wallet) => {
      // Simple sum - in reality you'd need exchange rates
      const balance = parseFloat(wallet.balanceString);
      return total + (isNaN(balance) ? 0 : balance);
    }, 0);
  };

  const getWalletTypeCount = (type: string) => {
    return wallets.filter((w) => w.walletType === type).length;
  };

  if (loading && wallets.length === 0) {
    return (
      <div className="min-h-screen bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="text-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Loading wallets...</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">
                Wallet Dashboard
              </h1>
              <p className="text-gray-600 mt-1">
                Manage your BitGo wallets and transfers
              </p>
            </div>
            <div className="flex space-x-3">
              <Button
                variant="outline"
                onClick={handleDiscoverWallets}
                disabled={loading}
              >
                {loading ? "Discovering..." : "Discover Wallets"}
              </Button>
              <Button>Create Wallet</Button>
            </div>
          </div>
        </div>

        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Total Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{wallets.length}</div>
              <p className="text-sm text-gray-500 mt-1">
                {wallets.filter((w) => w.isActive).length} active
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Warm Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {getWalletTypeCount("custodial")}
              </div>
              <Badge variant="default" className="mt-2">
                Ready to use
              </Badge>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Cold Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {getWalletTypeCount("cold")}
              </div>
              <Badge variant="secondary" className="mt-2">
                High security
              </Badge>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">
                Hot Wallets
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {getWalletTypeCount("hot")}
              </div>
              <Badge variant="warning" className="mt-2">
                Operations
              </Badge>
            </CardContent>
          </Card>
        </div>

        {/* Error Display */}
        {error && (
          <Card className="mb-6 border-red-200 bg-red-50">
            <CardContent className="pt-6">
              <div className="flex items-center space-x-2">
                <span className="text-red-600">⚠️</span>
                <p className="text-red-700">{error}</p>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setError(null)}
                  className="ml-auto"
                >
                  Dismiss
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Wallets Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {wallets.map((wallet) => (
            <WalletCard
              key={wallet.id}
              wallet={wallet}
              onViewDetails={handleViewDetails}
              onCreateTransfer={handleCreateTransfer}
              onSyncBalance={handleSyncBalance}
            />
          ))}
        </div>

        {/* Empty State */}
        {wallets.length === 0 && !loading && (
          <Card className="text-center py-12">
            <CardContent>
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
                    d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
                  />
                </svg>
              </div>
              <h3 className="text-xl font-medium text-gray-900 mb-2">
                No wallets found
              </h3>
              <p className="text-gray-500 mb-6">
                Discover wallets from BitGo or create a new wallet to get
                started.
              </p>
              <div className="flex justify-center space-x-4">
                <Button variant="outline" onClick={handleDiscoverWallets}>
                  Discover Wallets
                </Button>
                <Button>Create Wallet</Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Loading Overlay for Sync */}
        {syncing && (
          <div className="fixed inset-0 bg-black bg-opacity-25 flex items-center justify-center z-50">
            <Card className="p-6">
              <div className="flex items-center space-x-3">
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600"></div>
                <p>Syncing wallet balance...</p>
              </div>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
}
