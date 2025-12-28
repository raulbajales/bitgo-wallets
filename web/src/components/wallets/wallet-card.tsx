import React from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatCurrency, formatTimeAgo, truncateAddress } from "@/lib/utils";

export interface Wallet {
  id: string;
  bitgoWalletId: string;
  label: string;
  coin: string;
  walletType: "custodial" | "hot" | "cold";
  balanceString: string;
  confirmedBalanceString: string;
  spendableBalanceString: string;
  isActive: boolean;
  frozen: boolean;
  tags?: string[];
  createdAt: string;
  updatedAt: string;
}

interface WalletCardProps {
  wallet: Wallet;
  onViewDetails: (wallet: Wallet) => void;
  onCreateTransfer: (wallet: Wallet) => void;
  onSyncBalance: (wallet: Wallet) => void;
}

export function WalletCard({
  wallet,
  onViewDetails,
  onCreateTransfer,
  onSyncBalance,
}: WalletCardProps) {
  const getWalletTypeVariant = (type: string) => {
    switch (type) {
      case "custodial":
        return "default";
      case "hot":
        return "warning";
      case "cold":
        return "secondary";
      default:
        return "outline";
    }
  };

  const getWalletTypeLabel = (type: string) => {
    switch (type) {
      case "custodial":
        return "Warm";
      case "hot":
        return "Hot";
      case "cold":
        return "Cold";
      default:
        return type;
    }
  };

  return (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="flex-shrink-0">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white font-bold text-sm">
                {wallet.coin.toUpperCase()}
              </div>
            </div>
            <div>
              <CardTitle className="text-lg">{wallet.label}</CardTitle>
              <CardDescription>
                {truncateAddress(wallet.bitgoWalletId)}
              </CardDescription>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <Badge variant={getWalletTypeVariant(wallet.walletType)}>
              {getWalletTypeLabel(wallet.walletType)}
            </Badge>
            {wallet.frozen && <Badge variant="destructive">Frozen</Badge>}
            {!wallet.isActive && <Badge variant="outline">Inactive</Badge>}
          </div>
        </div>
      </CardHeader>

      <CardContent>
        <div className="space-y-4">
          {/* Balance Information */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-3 bg-gray-50 rounded-lg">
              <div className="text-sm text-gray-500 mb-1">Total Balance</div>
              <div className="text-lg font-semibold">
                {formatCurrency(wallet.balanceString, wallet.coin)}
              </div>
            </div>
            <div className="text-center p-3 bg-gray-50 rounded-lg">
              <div className="text-sm text-gray-500 mb-1">Confirmed</div>
              <div className="text-lg font-semibold">
                {formatCurrency(wallet.confirmedBalanceString, wallet.coin)}
              </div>
            </div>
            <div className="text-center p-3 bg-green-50 rounded-lg">
              <div className="text-sm text-green-600 mb-1">Spendable</div>
              <div className="text-lg font-semibold text-green-700">
                {formatCurrency(wallet.spendableBalanceString, wallet.coin)}
              </div>
            </div>
          </div>

          {/* Tags */}
          {wallet.tags && wallet.tags.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {wallet.tags.map((tag, index) => (
                <Badge key={index} variant="outline" className="text-xs">
                  {tag}
                </Badge>
              ))}
            </div>
          )}

          {/* Metadata */}
          <div className="text-sm text-gray-500 space-y-1">
            <div>Created: {formatTimeAgo(new Date(wallet.createdAt))}</div>
            <div>Updated: {formatTimeAgo(new Date(wallet.updatedAt))}</div>
          </div>

          {/* Actions */}
          <div className="flex space-x-2 pt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onViewDetails(wallet)}
              className="flex-1"
            >
              View Details
            </Button>
            <Button
              size="sm"
              onClick={() => onCreateTransfer(wallet)}
              disabled={wallet.frozen || !wallet.isActive}
              className="flex-1"
            >
              Send Transfer
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onSyncBalance(wallet)}
              title="Sync Balance"
            >
              🔄
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
