import { useState } from 'react'

function App() {
  const [count, setCount] = useState(0)

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center">
      <div className="bg-white p-8 rounded-lg shadow-md">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">s3c</h1>
        <p className="text-gray-600 mb-4">S3 Client - Frontend will be implemented with hash routing</p>
        <button 
          onClick={() => setCount((count) => count + 1)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          count is {count}
        </button>
      </div>
    </div>
  )
}

export default App