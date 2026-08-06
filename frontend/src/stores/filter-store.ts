import { create } from "zustand"
import { persist } from "zustand/middleware"

interface FilterState {
  // Services page
  servicesStatus: string
  servicesAlert: string
  servicesConfig: string
  setServicesStatus: (v: string) => void
  setServicesAlert: (v: string) => void
  setServicesConfig: (v: string) => void

  // Nodes page
  nodesRole: string
  nodesStatus: string
  setNodesRole: (v: string) => void
  setNodesStatus: (v: string) => void

  // Recommendations page
  recConfidence: string
  recEvents: string[]
  recStatus: string
  setRecConfidence: (v: string) => void
  setRecEvents: (v: string[]) => void
  setRecStatus: (v: string) => void

  // ServiceDetail page
  serviceContainerFilter: string
  setServiceContainerFilter: (v: string) => void

  // Tasks page (Fase 7)
  tasksStatus: string
  tasksService: string
  setTasksStatus: (v: string) => void
  setTasksService: (v: string) => void

  // Agents page (Fase 7)
  agentsStatus: string
  setAgentsStatus: (v: string) => void

  // Alerts page
  alertsType: string
  alertsSeverity: string
  setAlertsType: (v: string) => void
  setAlertsSeverity: (v: string) => void

  // Schedules page
  schedulesStatus: string
  schedulesLogSource: string
  schedulesSearch: string
  schedulesTab: string
  setSchedulesStatus: (v: string) => void
  setSchedulesLogSource: (v: string) => void
  setSchedulesSearch: (v: string) => void
  setSchedulesTab: (v: string) => void
}

export const useFilterStore = create<FilterState>()(
  persist(
    (set) => ({
      servicesStatus: "all",
      servicesAlert: "all",
      servicesConfig: "all",
      setServicesStatus: (v) => set({ servicesStatus: v }),
      setServicesAlert: (v) => set({ servicesAlert: v }),
      setServicesConfig: (v) => set({ servicesConfig: v }),

      nodesRole: "all",
      nodesStatus: "all",
      setNodesRole: (v) => set({ nodesRole: v }),
      setNodesStatus: (v) => set({ nodesStatus: v }),

      recConfidence: "all",
      recEvents: [],
      recStatus: "all",
      setRecConfidence: (v) => set({ recConfidence: v }),
      setRecEvents: (v) => set({ recEvents: v }),
      setRecStatus: (v) => set({ recStatus: v }),

      serviceContainerFilter: "all",
      setServiceContainerFilter: (v) => set({ serviceContainerFilter: v }),

      tasksStatus: "all",
      tasksService: "all",
      setTasksStatus: (v) => set({ tasksStatus: v }),
      setTasksService: (v) => set({ tasksService: v }),

      agentsStatus: "all",
      setAgentsStatus: (v) => set({ agentsStatus: v }),

      alertsType: "all",
      alertsSeverity: "all",
      setAlertsType: (v) => set({ alertsType: v }),
      setAlertsSeverity: (v) => set({ alertsSeverity: v }),

      schedulesStatus: "all",
      schedulesLogSource: "all",
      schedulesSearch: "",
      schedulesTab: "pending",
      setSchedulesStatus: (v) => set({ schedulesStatus: v }),
      setSchedulesLogSource: (v) => set({ schedulesLogSource: v }),
      setSchedulesSearch: (v) => set({ schedulesSearch: v }),
      setSchedulesTab: (v) => set({ schedulesTab: v }),
    }),
    { name: "resma-filters" }
  )
)
