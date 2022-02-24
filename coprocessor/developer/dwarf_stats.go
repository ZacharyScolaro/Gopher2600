// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package developer

// Load records the frame (or current) load as well as the average and
// maximum load.
type Load struct {
	Frame   float32
	Average float32
	Max     float32
}

// Stats records the cycle count over time and can be used to the frame
// (or current) load as well as average and maximum load.
//
// The actual percentage values are accessed through the OverSource and
// OverFunction fields. These fields provide the necessary scale by which
// the load is measured.
//
// The validity of the OverSource and OverFunction fields depends on context.
// For instance, for the SourceFunction type, the corresponding OverFunction
// field is invalid. For the Source type meanwhile, neither field is valid.
//
// For the SourceLine type however, both OverSource and OverFunction can be
// used to provide a different scaling to the load values.
type Stats struct {
	OverSource   Load
	OverFunction Load

	cumulativeCount float32
	numFrames       float32
	avgCount        float32

	frameCount float32
	count      float32
}

// IsValid returns true if the statistics have ever been updated. ie. the
// source associated with this statistic has ever executed.
func (stats *Stats) IsValid() bool {
	return stats.cumulativeCount > 0
}

// update statistics, using source and function to update the Load values as
// appropriate.
func (stats *Stats) newFrame(source *Stats, function *Stats) {
	stats.cumulativeCount += stats.count
	stats.numFrames++
	stats.avgCount = stats.cumulativeCount / stats.numFrames

	stats.frameCount = stats.count
	stats.count = 0

	if function != nil {
		frameLoad := stats.frameCount / function.frameCount * 100
		stats.OverFunction.Frame = frameLoad

		stats.OverFunction.Average = stats.avgCount / function.avgCount * 100

		if frameLoad > stats.OverFunction.Max {
			stats.OverFunction.Max = frameLoad
		}
	}

	if source != nil {
		frameLoad := stats.frameCount / source.frameCount * 100
		stats.OverSource.Frame = frameLoad

		stats.OverSource.Average = stats.avgCount / source.avgCount * 100

		if frameLoad > stats.OverSource.Max {
			stats.OverSource.Max = frameLoad
		}
	}
}
