"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Loader2, AlertCircle, CheckCircle2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { checkinApi, type BoardingPass, type CheckInResult } from "@/lib/api-client";

export default function CheckInPnrPage() {
  const params = useParams<{ pnr: string }>();
  const router = useRouter();
  const pnr = String(params?.pnr || "").toUpperCase();

  const [checkin, setCheckin] = useState<CheckInResult | null>(null);
  const [boardingPass, setBoardingPass] = useState<BoardingPass | null>(null);
  const [seat, setSeat] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const ci = await checkinApi.initiate(pnr);
        if (cancelled) return;
        setCheckin(ci);
      } catch {
        try {
          const bp = await checkinApi.boardingPass(pnr);
          if (cancelled) return;
          setBoardingPass(bp);
        } catch (e2) {
          if (!cancelled) setError(e2 instanceof Error ? e2.message : "Check-in failed");
        }
      }
    })();
    return () => { cancelled = true; };
  }, [pnr]);

  const assignSeat = async () => {
    if (!seat.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const bp = await checkinApi.assignSeat(pnr, seat.trim().toUpperCase());
      setBoardingPass(bp);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Seat assignment failed");
    } finally {
      setBusy(false);
    }
  };

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-950">
        <Card className="w-full max-w-md">
          <CardContent className="p-8 text-center">
            <AlertCircle className="mx-auto mb-4 h-12 w-12 text-red-400" />
            <h2 className="mb-2 text-xl font-bold">Check-in unavailable</h2>
            <p className="mb-6 text-sm text-gray-500">{error}</p>
            <Button variant="outline" className="w-full" onClick={() => router.push("/check-in")}>Try Another PNR</Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!checkin && !boardingPass) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-8 dark:bg-gray-950">
      <div className="w-full max-w-md space-y-4">
        {checkin && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400"><CheckCircle2 className="h-5 w-5" /> Checked In</CardTitle>
              <CardDescription>PNR {pnr} is checked in online</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div><div className="text-gray-500">Baggage</div><div className="font-medium">{checkin.baggage_count} pcs</div></div>
                <div><div className="text-gray-500">Weight</div><div className="font-medium">{checkin.baggage_weight} kg</div></div>
              </div>
            </CardContent>
          </Card>
        )}

        {!boardingPass && (
          <Card>
            <CardHeader><CardTitle>Choose Your Seat</CardTitle><CardDescription>Pick a seat to receive your boarding pass</CardDescription></CardHeader>
            <CardContent className="space-y-4">
              {error && <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">{error}</div>}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-600 dark:text-gray-400">Seat number</label>
                <input type="text" value={seat} onChange={e => setSeat(e.target.value.toUpperCase())} placeholder="12A" className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" />
              </div>
              <Button className="btn-gradient w-full" onClick={assignSeat} disabled={busy || !seat.trim()}>
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "Assign Seat"}
              </Button>
            </CardContent>
          </Card>
        )}

        {boardingPass && (
          <Card>
            <CardHeader><CardTitle>Boarding Pass</CardTitle><CardDescription>{boardingPass.flight_number} · {boardingPass.origin} → {boardingPass.destination}</CardDescription></CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div><div className="text-gray-500">Passenger</div><div className="font-medium">#{boardingPass.passenger_id}</div></div>
                <div><div className="text-gray-500">Seat</div><div className="font-semibold text-lg">{boardingPass.seat_number}</div></div>
                <div><div className="text-gray-500">Gate</div><div className="font-medium">{boardingPass.gate}</div></div>
                <div><div className="text-gray-500">Boarding</div><div className="font-medium">{boardingPass.boarding_group}</div></div>
              </div>
              <div className="rounded-lg bg-gray-100 p-4 text-center dark:bg-gray-800">
                <div className="font-mono text-2xl tracking-widest text-gray-800 dark:text-gray-200">{boardingPass.qr_code_data || "••••••"}</div>
                <div className="mt-1 text-xs text-gray-500">Scan at the gate</div>
              </div>
              <div className="flex gap-3">
                <Button variant="outline" className="flex-1" onClick={() => router.push(`/boarding-pass/${pnr}`)}>View Boarding Pass</Button>
                <Button variant="outline" className="flex-1" onClick={() => router.push("/")}>Done</Button>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
