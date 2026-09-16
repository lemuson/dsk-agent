import { api } from './client';
import { ResidentialComplex, Building, Apartment, ConstructionProgress } from '../types';

export const apartmentsApi = {

  getComplexes: async (): Promise<ResidentialComplex[]> => {
    const res = await api.get<ResidentialComplex[]>('/complexes', false);
    return Array.isArray(res) ? res : [];
  },

  getComplex: async (id: number): Promise<ResidentialComplex> => {
    return api.get<ResidentialComplex>(`/complexes/${id}`, false);
  },

  getBuildingsByComplex: async (complexId: number): Promise<Building[]> => {
    const res = await api.get<Building[]>(`/complexes/${complexId}/buildings`, false);
    return Array.isArray(res) ? res : [];
  },

  getBuilding: async (id: number): Promise<Building> => {
    return api.get<Building>(`/buildings/${id}`, false);
  },

  getApartmentsByBuilding: async (buildingId: number): Promise<Apartment[]> => {
    const res = await api.get<Apartment[]>(`/buildings/${buildingId}/apartments`, false);
    return Array.isArray(res) ? res : [];
  },

  getApartment: async (id: number): Promise<Apartment> => {
    return api.get<Apartment>(`/apartments/${id}`, false);
  },

  getProgressByBuilding: async (buildingId: number): Promise<ConstructionProgress[]> => {
    const res = await api.get<ConstructionProgress[]>(`/buildings/${buildingId}/progress`, false);
    return Array.isArray(res) ? res : [];
  },

  getAncillaryUnits: async (buildingId: number): Promise<any[]> => {
    const res = await api.get<any[]>(`/buildings/${buildingId}/ancillary-units`, false);
    return Array.isArray(res) ? res : [];
  },

  getAllAvailableApartments: async (): Promise<{
    apartments: Apartment[];
    complexes: ResidentialComplex[];
    buildings: Building[];
  }> => {
    const complexes = await apartmentsApi.getComplexes();
    const allBuildings: Building[] = [];
    const allApartments: Apartment[] = [];

    if (Array.isArray(complexes)) {
      for (const complex of complexes) {
        if (!complex || typeof complex.id !== 'number') continue;
        try {
          const buildings = await apartmentsApi.getBuildingsByComplex(complex.id);
          if (Array.isArray(buildings)) {
            allBuildings.push(...buildings);

            for (const building of buildings) {
              if (!building || typeof building.id !== 'number') continue;
              try {
                const apts = await apartmentsApi.getApartmentsByBuilding(building.id);
                if (Array.isArray(apts)) {
                  const enrichedApts = apts.map((apt) => ({
                    ...apt,
                    building,
                    complex,
                  }));
                  allApartments.push(...enrichedApts);
                }
              } catch (e) {
                console.warn(`Failed to fetch apartments for building ${building.id}`, e);
              }
            }
          }
        } catch (e) {
          console.warn(`Failed to fetch buildings for complex ${complex.id}`, e);
        }
      }
    }

    return {
      apartments: allApartments,
      complexes: Array.isArray(complexes) ? complexes : [],
      buildings: allBuildings,
    };
  },
};
