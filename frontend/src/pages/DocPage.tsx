import { useParams } from 'react-router-dom'

export function DocPage() {
  const { id } = useParams<{ id: string }>()
  return <h1>Doc {id ?? ':id'}</h1>
}
