"use client";

import { useCallback, useMemo } from "react";
import { cn } from "@/lib/utils";

type CabinCode = string;
type SeatPosition = string;
type CabinSection = { code: CabinCode; name: string; rows: { from: number; to: number }; cols: string[] };
interface Seat { id: number; row: number; col: string; position: SeatPosition; cabin: CabinCode; is_available: boolean; is_emergency_exit: boolean; is_extra_legroom: boolean; }
interface SeatMapData { aircraft: string; cabins: CabinSection[]; seats: Seat[]; legend: Record<string, { color: string; label: string }>; }

const CABIN_LABELS: Record<string, string> = { Y: "Economy", W: "Premium Economy", J: "Business", F: "First" };
const STATUS_FILLS: Record<string, string> = {
  AVAILABLE: "fill-white dark:fill-gray-900",
  SELECTED: "fill-blue-500 dark:fill-blue-400",
  OCCUPIED: "fill-gray-300 dark:fill-gray-700",
  EXIT: "fill-amber-100 dark:fill-amber-900/30",
  EXTRALEG: "fill-purple-100 dark:fill-purple-900/30",
};
const CABIN_STROKES: Record<string, string> = { Y: "stroke-blue-200 dark:stroke-blue-800", J: "stroke-amber-200 dark:stroke-amber-800", W: "stroke-purple-200 dark:stroke-purple-800", F: "stroke-emerald-200 dark:stroke-emerald-800" };

interface SeatMapProps { data: SeatMapData; selectedSeats: string[]; onSelect: (seats: string[]) => void; maxSeats?: number; }

export function SeatMap({ data, selectedSeats, onSelect, maxSeats = 5 }: SeatMapProps) {
  const seatMap = useMemo(() => { const m = new Map<string, Seat>(); for (const s of data.seats) m.set(`${s.row}${s.col}`, s); return m; }, [data.seats]);

  const toggleSeat = useCallback((seatId: string) => {
    const seat = seatMap.get(seatId);
    if (!seat || !seat.is_available) return;
    const next = new Set(selectedSeats);
    if (next.has(seatId)) next.delete(seatId);
    else if (next.size < (maxSeats || 5)) next.add(seatId);
    onSelect(Array.from(next));
  }, [seatMap, selectedSeats, maxSeats, onSelect]);

  const getStatus = (seat: Seat) => {
    if (seat.is_emergency_exit) return "EXIT";
    if (seat.is_extra_legroom) return "EXTRALEG";
    if (selectedSeats.includes(`${seat.row}${seat.col}`)) return "SELECTED";
    if (!seat.is_available) return "OCCUPIED";
    return "AVAILABLE";
  };

  const groupedByCabin = useMemo(() => {
    const g: Record<string, Seat[]> = {};
    for (const cabin of data.cabins) g[cabin.code] = data.seats.filter(s => s.cabin === cabin.code);
    return g;
  }, [data]);

  return (
    <div className="flex flex-col items-center gap-6">
      <svg width="180" height="35" viewBox="0 0 180 35" className="mb-2" aria-hidden>
        <path d="M60 35 L90 3 L120 35" fill="none" stroke="#e5e7eb" className="dark:stroke-gray-600" strokeWidth="2" strokeLinejoin="round" />
      </svg>

      {data.cabins.map(cabin => {
        const cabinSeats = groupedByCabin[cabin.code] || [];
        if (!cabinSeats.length) return null;
        const rows = Array.from({ length: cabin.rows.to - cabin.rows.from + 1 }, (_, i) => cabin.rows.from + i);
        const cols = cabin.cols;
        const mid = Math.ceil(cols.length / 2);

        return (
          <div key={cabin.code} className="w-full max-w-md">
            <div className="mb-2 flex items-center justify-between px-2">
              <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">{CABIN_LABELS[cabin.code] || cabin.name}</span>
              <span className="text-xs text-gray-500 dark:text-gray-400">Rows {cabin.rows.from}–{cabin.rows.to}</span>
            </div>
            <div className="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
              <svg width="100%" viewBox={`0 0 ${cols.length * 36 + 30} ${rows.length * 40 + 15}`} className="mx-auto">
                {cols.map((col, ci) => (
                  <text key={col} x={ci < mid ? 14 + ci * 36 : 26 + ci * 36} y={10} textAnchor="middle" className="fill-gray-400 text-[9px] font-medium">{col}</text>
                ))}
                {cols.length > 4 && <line x1={mid * 36 + 16} y1={16} x2={mid * 36 + 16} y2={rows.length * 40 + 10} stroke="#e5e7eb" className="dark:stroke-gray-700" strokeWidth="0.5" strokeDasharray="3 2" />}
                {rows.map(rowNum => cols.map((col, ci) => {
                  const seat = cabinSeats.find(s => s.row === rowNum && s.col === col);
                  if (!seat) return null;
                  const isLeft = ci < mid;
                  const x = isLeft ? 14 + ci * 36 : 26 + ci * 36;
                  const y = 20 + (rowNum - cabin.rows.from) * 40;
                  const status = getStatus(seat);
                  const selectable = status === "AVAILABLE" || status === "SELECTED" || (status === "EXIT" && !seat.is_available);
                  const fillCls = status === "EXIT" && !seat.is_available ? STATUS_FILLS["EXIT"] : STATUS_FILLS[status];

                  return (
                    <g key={`${seat.row}${seat.col}`}>
                      <rect x={x - 10} y={y - 8} width={20} height={24} rx={3}
                        className={cn(CABIN_STROKES[seat.cabin] || CABIN_STROKES.Y, fillCls, selectable && "cursor-pointer")}
                        strokeWidth={selectedSeats.includes(`${seat.row}${seat.col}`) ? 2 : 1}
                        onClick={() => selectable && toggleSeat(`${seat.row}${seat.col}`)}
                      />
                      <text x={x} y={y + 5} textAnchor="middle" className={cn("text-[8px] font-medium", selectedSeats.includes(`${seat.row}${seat.col}`) ? "fill-white" : "fill-gray-600 dark:fill-gray-300")}>{seat.row}{seat.col}</text>
                    </g>
                  );
                }))}
              </svg>
            </div>
          </div>
        );
      })}

      <div className="flex flex-wrap justify-center gap-4 text-xs">
        {data.legend && Object.entries(data.legend).map(([key, val]) => (
          <div key={key} className="flex items-center gap-1.5">
            <div className="h-3 w-3 rounded-sm border border-gray-300 dark:border-gray-600" style={{ backgroundColor: val.color }} />
            <span className="text-gray-600 dark:text-gray-400">{val.label}</span>
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-800/50 w-full max-w-md text-sm">
        <span className="text-gray-600 dark:text-gray-400">
          {selectedSeats.length === 0 ? `Select up to ${maxSeats} seats` : `${selectedSeats.length} of ${maxSeats} seats selected`}
        </span>
        {selectedSeats.length > 0 && <span className="font-medium text-gray-900 dark:text-gray-100">{selectedSeats.join(", ")}</span>}
      </div>
    </div>
  );
}