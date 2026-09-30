import type {IAbstract} from './IAbstract'

export interface IAdminOverviewShares {
	linkShares: number
	teamShares: number
	userShares: number
}

export interface IAdminOverview extends IAbstract {
	users: number
	projects: number
	tasks: number
	teams: number
	shares: IAdminOverviewShares
}
