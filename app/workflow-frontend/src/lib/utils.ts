import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import moment from "moment"
import { TaskStatus } from "@/services/worfklows/workflows.schema";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * convert date into a readable date
 * @param isoDateString 
 * @param formatString 
 * @returns 
 */
export const readableDate = (isoDateString: string, formatString: string = 'MMMM Do YYYY, h:mm:ss a') => {
  return moment(isoDateString).format(formatString)
}


export const borderStatusIndicator = (status: TaskStatus) => {
    if (status === "in_progress") return "border-yellow-700"
    if (status === "success") return "border-green-700"
    if (status === "failed") return "border-red-700"
  }