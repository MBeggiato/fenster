import AbstractModel from './abstractModel'
import type {IAdminOverview, IAdminOverviewShares} from '@/modelTypes/IAdminOverview'

export default class AdminOverviewModel extends AbstractModel<IAdminOverview> implements IAdminOverview {
	users = 0
	projects = 0
	tasks = 0
	teams = 0
	shares: IAdminOverviewShares = {
		linkShares: 0,
		teamShares: 0,
		userShares: 0,
	}

	constructor(data: Partial<IAdminOverview> = {}) {
		super()
		this.assignData(data)
	}
}
