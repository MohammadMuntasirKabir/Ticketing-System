"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Loader2, AlertCircle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { checkinApi, type BoardingPass } from "@/lib/api-client";

export default function BoardingPassPage() {
  const params = useParams<{ pnr: string }>();
  const router = useRouter();
  const pnr = String(params?.pnr || "").toUpperCase();

  const [bp, setBp] = useState<BoardingPass | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    checkinApi.boardingPass(pnr)
      .then(data => { if (!cancelled) setBp(data); })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load boarding pass"); });
    return () => { cancelled = true; };
  }, [pnr]);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-950">
        <Card className="w-full max-w-md">
          <CardContent className="p-8 text-center">
            <AlertCircle className="mx-auto mb-4 h-12 w-12 text-red-400" />
            <h2 className="mb-2 text-xl font-bold">No boarding pass</h2>
            <p className="mb-6 text-sm text-gray-500">{error}</p>
            <Button variant="outline" className="w-full" onClick={() => router.push("/check-in")}>Check In</Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!bp) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-8 dark:bg-gray-950">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center">
          <h1 className="text-2xl font-bold">Boarding Pass</h1>
          <p className="text-sm text-gray-500">PNR {pnr}</p>
        </div>
        <Card className="overflow-hidden">
          <div className="bg-brand-gradient px-6 py-3 text-white">
            <div className="flex items-center justify-between">
              <span className="text-lg font-bold">{bp.flight_number}</span>
              <span className="text-sm">{new Date(bp.boarding_time).toLocaleString()}</span>
            </div>
          </div>
          <CardContent className="p-6">
            <div className="mb-6 flex items-center justify-between">
              <div>
                <div className="text-2xl font-bold">{bp.origin}</div>
                <div className="text-xs text-gray-500">From</div>
              </div>
              <div className="flex-1 border-t-2 border-dashed border-gray-300 mx-4" />
              <div className="text-right">
                <div className="text-2xl font-bold">{bp.destination}</div>
                <div className="text-xs text-gray-500">To</div>
              </div>
            </div>
            <div className="grid grid-cols-3 gap-3 text-center">
              <div className="rounded-lg bg-gray-100 p-3 dark:bg-gray-800">
                <div className="text-xs text-gray-500">Seat</div>
                <div className="text-lg font-bold">{bp.seat_number}</div>
              </div>
              <div className="rounded-lg bg-gray-100 p-3 dark:bg-gray-800">
                <div className="text-xs text-gray-500">Gate</div>
                <div className="text-lg font-bold">{bp.gate}</div>
              </div>
              <div className="rounded-lg bg-gray-100 p-3 dark:bg-gray-800">
                <div className="text-xs text-gray-500">Group</div>
                <div className="text-lg font-bold">{bp.boarding_group}</div>
              </div>
            </div>
            <div className="mt-4 rounded-lg bg-gray-100 p-4 text-center dark:bg-gray-800">
              <div className="font-mono text-2xl tracking-widest text-gray-800 dark:text-gray-200">{bp.qr_code_data || "••••••"}</div>
              <div className="mt-1 text-xs text-gray-500">Scan at the gate</div>
            </div>
          </CardContent>
        </Card>
        <div className="mt-6 flex justify-center">
          <Button variant="outline" onClick={() => router.push("/")}>Done</Button>
        </div>
      </div>
    </div>
  );
}
