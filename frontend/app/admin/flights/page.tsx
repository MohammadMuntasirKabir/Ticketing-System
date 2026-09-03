"use client";

import { useEffect, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Plus, CalendarRange, Loader2 } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { adminApi, type FlightInstance } from "@/lib/api-client";
import { getToken } from "@/lib/auth";

function statusBadge(status: string) {
  const tone =
    status === "SCHEDULED" ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
    : status === "BOARDING" ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
    : status === "DELAYED" ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
    : "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
  return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${tone}`}>{status}</span>;
}

export default function AdminFlightsPage() {
  const [flights, setFlights] = useState<FlightInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const res = await adminApi.flights.list(getToken());
        setFlights(res.data);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load flights");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold tracking-tight">Flights</h1><p className="text-sm text-gray-500 dark:text-gray-400">Manage flight schedules and instances</p></div>
        <div className="flex gap-3">
          <Button variant="outline"><CalendarRange className="mr-2 h-4 w-4" /> Materialize</Button>
          <Button><Plus className="mr-2 h-4 w-4" /> New Flight</Button>
        </div>
      </div>
      {error && <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-400">{error}</div>}
      <Card><CardContent className="p-0">
        {loading ? (
          <div className="flex items-center justify-center gap-2 py-16 text-gray-500"><Loader2 className="h-5 w-5 animate-spin" /> Loading flights…</div>
        ) : (
          <Table>
            <TableHeader><TableRow><TableHead>ID</TableHead><TableHead>Schedule</TableHead><TableHead>Departure</TableHead><TableHead>Arrival</TableHead><TableHead>Status</TableHead><TableHead>Delay</TableHead></TableRow></TableHeader>
            <TableBody>
              {flights.map(f => (
                <TableRow key={f.id}>
                  <TableCell className="font-semibold">#{f.id}</TableCell>
                  <TableCell>{f.schedule_id}</TableCell>
                  <TableCell>{new Date(f.departure_time).toLocaleString()}</TableCell>
                  <TableCell>{new Date(f.arrival_time).toLocaleString()}</TableCell>
                  <TableCell>{statusBadge(f.status)}</TableCell>
                  <TableCell>{f.delay_minutes ? `${f.delay_minutes}m` : "—"}</TableCell>
                </TableRow>
              ))}
              {flights.length === 0 && (
                <TableRow><TableCell colSpan={6} className="py-8 text-center text-gray-500">No flights found.</TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </CardContent></Card>
    </div>
  );
}
